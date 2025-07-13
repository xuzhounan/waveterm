// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package wconfig

import (
	"log"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/wavetermdev/waveterm/pkg/panichandler"
	"github.com/wavetermdev/waveterm/pkg/wavebase"
	"github.com/wavetermdev/waveterm/pkg/wps"
)

var instance *Watcher
var once sync.Once

type Watcher struct {
	initialized bool
	watcher     *fsnotify.Watcher
	mutex       sync.Mutex
	fullConfig  FullConfigType
}

type WatcherUpdate struct {
	FullConfig FullConfigType `json:"fullconfig"`
}

// GetWatcher returns the singleton instance of the Watcher
func GetWatcher() *Watcher {
	once.Do(func() {
		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			log.Printf("failed to create file watcher: %v", err)
			return
		}
		configDirAbsPath := wavebase.GetWaveConfigDir()
		instance = &Watcher{watcher: watcher}
		err = instance.watcher.Add(configDirAbsPath)
		const failedStr = "failed to add path %s to watcher: %v"
		if err != nil {
			log.Printf(failedStr, configDirAbsPath, err)
		}

		subdirs := GetConfigSubdirs()
		for _, dir := range subdirs {
			err = instance.watcher.Add(dir)
			if err != nil {
				log.Printf(failedStr, dir, err)
			}
		}
	})
	return instance
}

func (w *Watcher) Start() {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	log.Printf("starting file watcher\n")
	w.initialized = true
	w.sendInitialValues()

	go func() {
		defer func() {
			panichandler.PanicHandler("filewatcher:Start", recover())
		}()
		for {
			select {
			case event, ok := <-w.watcher.Events:
				if !ok {
					return
				}
				w.handleEvent(event)
			case err, ok := <-w.watcher.Errors:
				if !ok {
					return
				}
				log.Println("watcher error:", err)
			}
		}
	}()
}

// for initial values, exit on first error
func (w *Watcher) sendInitialValues() error {
	w.fullConfig = ReadFullConfig()
	message := WatcherUpdate{
		FullConfig: w.fullConfig,
	}
	w.broadcast(message)
	return nil
}

func (w *Watcher) Close() {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	if w.watcher != nil {
		w.watcher.Close()
		w.watcher = nil
		log.Println("file watcher closed")
	}
}

func (w *Watcher) broadcast(message WatcherUpdate) {
	// send to frontend
	wps.Broker.Publish(wps.WaveEvent{
		Event: wps.Event_Config,
		Data:  message,
	})
}

func (w *Watcher) GetFullConfig() FullConfigType {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	return w.fullConfig
}

func (w *Watcher) handleEvent(event fsnotify.Event) {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	fileName := filepath.ToSlash(event.Name)
	if event.Op == fsnotify.Chmod {
		return
	}
	if !isValidSubSettingsFileName(fileName) {
		return
	}
	w.handleSettingsFileEvent(event, fileName)
}

var validFileRe = regexp.MustCompile(`^[a-zA-Z0-9_@.-]+\.json$`)

func isValidSubSettingsFileName(fileName string) bool {
	if filepath.Ext(fileName) != ".json" {
		return false
	}
	baseName := filepath.Base(fileName)
	return validFileRe.MatchString(baseName)
}

func (w *Watcher) handleSettingsFileEvent(event fsnotify.Event, fileName string) {
	// Check if this is a new workspace directory creation
	if event.Op&fsnotify.Create != 0 && strings.Contains(fileName, "/workspaces/") && filepath.Ext(fileName) == ".json" {
		// Extract workspace ID from path
		pathParts := strings.Split(fileName, "/")
		for i, part := range pathParts {
			if part == "workspaces" && i+1 < len(pathParts) {
				workspaceId := pathParts[i+1]
				w.addWorkspaceWatcher(workspaceId)
				break
			}
		}
	}
	
	fullConfig := ReadFullConfig()
	w.fullConfig = fullConfig
	w.broadcast(WatcherUpdate{FullConfig: w.fullConfig})
}

// addWorkspaceWatcher adds a new workspace directory to the file watcher
func (w *Watcher) addWorkspaceWatcher(workspaceId string) {
	configDirAbsPath := wavebase.GetWaveConfigDir()
	workspaceDir := filepath.Join(configDirAbsPath, "workspaces", workspaceId)
	
	err := w.watcher.Add(workspaceDir)
	if err != nil {
		log.Printf("failed to add workspace %s to watcher: %v", workspaceId, err)
	} else {
		log.Printf("added workspace %s to file watcher", workspaceId)
	}
}

// AddWorkspaceWatcher adds a new workspace directory to the file watcher (public method)
func (w *Watcher) AddWorkspaceWatcher(workspaceId string) {
	if w == nil || w.watcher == nil {
		return
	}
	w.mutex.Lock()
	defer w.mutex.Unlock()
	w.addWorkspaceWatcher(workspaceId)
}
