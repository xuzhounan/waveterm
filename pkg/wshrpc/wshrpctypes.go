// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

// types and methods for wsh rpc calls
package wshrpc

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"os"
	"reflect"

	"github.com/wavetermdev/waveterm/pkg/ijson"
	"github.com/wavetermdev/waveterm/pkg/telemetry/telemetrydata"
	"github.com/wavetermdev/waveterm/pkg/util/iochan/iochantypes"
	"github.com/wavetermdev/waveterm/pkg/vdom"
	"github.com/wavetermdev/waveterm/pkg/waveobj"
	"github.com/wavetermdev/waveterm/pkg/wconfig"
	"github.com/wavetermdev/waveterm/pkg/wps"
)

const (
	// MaxFileSize is the maximum file size that can be read
	MaxFileSize = 50 * 1024 * 1024 // 50M
	// MaxDirSize is the maximum number of entries that can be read in a directory
	MaxDirSize = 1024
	// FileChunkSize is the size of the file chunk to read
	FileChunkSize = 64 * 1024
	// DirChunkSize is the size of the directory chunk to read
	DirChunkSize = 128
)

const LocalConnName = "local"

const (
	RpcType_Call             = "call"             // single response (regular rpc)
	RpcType_ResponseStream   = "responsestream"   // stream of responses (streaming rpc)
	RpcType_StreamingRequest = "streamingrequest" // streaming request
	RpcType_Complex          = "complex"          // streaming request/response
)

const (
	CreateBlockAction_Replace    = "replace"
	CreateBlockAction_SplitUp    = "splitup"
	CreateBlockAction_SplitDown  = "splitdown"
	CreateBlockAction_SplitLeft  = "splitleft"
	CreateBlockAction_SplitRight = "splitright"
)

// TODO generate these constants from the interface
const (
	Command_Authenticate      = "authenticate"      // special
	Command_AuthenticateToken = "authenticatetoken" // special
	Command_Dispose           = "dispose"           // special (disposes of the route, for multiproxy only)
	Command_RouteAnnounce     = "routeannounce"     // special (for routing)
	Command_RouteUnannounce   = "routeunannounce"   // special (for routing)
	Command_Message           = "message"
	Command_GetMeta           = "getmeta"
	Command_SetMeta           = "setmeta"
	Command_SetView           = "setview"
	Command_ControllerInput   = "controllerinput"
	Command_ControllerRestart = "controllerrestart"
	Command_ControllerStop    = "controllerstop"
	Command_ControllerResync  = "controllerresync"
	Command_Mkdir             = "mkdir"
	Command_ResolveIds        = "resolveids"
	Command_BlockInfo         = "blockinfo"
	Command_CreateBlock       = "createblock"
	Command_DeleteBlock       = "deleteblock"

	Command_FileWrite           = "filewrite"
	Command_FileRead            = "fileread"
	Command_FileReadStream      = "filereadstream"
	Command_FileMove            = "filemove"
	Command_FileCopy            = "filecopy"
	Command_FileStreamTar       = "filestreamtar"
	Command_FileAppend          = "fileappend"
	Command_FileAppendIJson     = "fileappendijson"
	Command_FileJoin            = "filejoin"
	Command_FileShareCapability = "filesharecapability"

	Command_EventPublish         = "eventpublish"
	Command_EventRecv            = "eventrecv"
	Command_EventSub             = "eventsub"
	Command_EventUnsub           = "eventunsub"
	Command_EventUnsubAll        = "eventunsuball"
	Command_EventReadHistory     = "eventreadhistory"
	Command_StreamTest           = "streamtest"
	Command_StreamWaveAi         = "streamwaveai"
	Command_StreamCpuData        = "streamcpudata"
	Command_Test                 = "test"
	Command_SetConfig                = "setconfig"
	Command_SetConnectionsConfig     = "connectionsconfig"
	Command_SetWorkspaceWidgetConfig    = "setworkspacewidgetconfig"
	Command_EnsureWorkspaceWidgetConfig = "ensureworkspacewidgetconfig"
	Command_GetFullConfig            = "getfullconfig"
	Command_RemoteStreamFile     = "remotestreamfile"
	Command_RemoteTarStream      = "remotetarstream"
	Command_RemoteFileInfo       = "remotefileinfo"
	Command_RemoteFileTouch      = "remotefiletouch"
	Command_RemoteWriteFile      = "remotewritefile"

	Command_RemoteFileDelete     = "remotefiledelete"
	Command_RemoteFileJoin       = "remotefilejoin"
	Command_WaveInfo             = "waveinfo"
	Command_WshActivity          = "wshactivity"
	Command_Activity             = "activity"
	Command_GetVar               = "getvar"
	Command_SetVar               = "setvar"
	Command_RemoteMkdir          = "remotemkdir"
	Command_RemoteGetInfo        = "remotegetinfo"
	Command_RemoteInstallRcfiles = "remoteinstallrcfiles"

	Command_ConnStatus       = "connstatus"
	Command_WslStatus        = "wslstatus"
	Command_ConnEnsure       = "connensure"
	Command_ConnReinstallWsh = "connreinstallwsh"
	Command_ConnConnect      = "connconnect"
	Command_ConnDisconnect   = "conndisconnect"
	Command_ConnList         = "connlist"
	Command_ConnListAWS      = "connlistaws"
	Command_WslList          = "wsllist"
	Command_WslDefaultDistro = "wsldefaultdistro"
	Command_DismissWshFail   = "dismisswshfail"
	Command_ConnUpdateWsh    = "updatewsh"

	Command_WorkspaceList = "workspacelist"

	Command_WebSelector      = "webselector"
	Command_Notify           = "notify"
	Command_FocusWindow      = "focuswindow"
	Command_GetUpdateChannel = "getupdatechannel"

	Command_VDomCreateContext   = "vdomcreatecontext"
	Command_VDomAsyncInitiation = "vdomasyncinitiation"
	Command_VDomRender          = "vdomrender"
	Command_VDomUrlRequest      = "vdomurlrequest"

	Command_AiSendMessage = "aisendmessage"
)

type RespOrErrorUnion[T any] struct {
	Response T
	Error    error
}

type WshRpcInterface interface {
	AuthenticateCommand(ctx context.Context, data string) (CommandAuthenticateRtnData, error)
	AuthenticateTokenCommand(ctx context.Context, data CommandAuthenticateTokenData) (CommandAuthenticateRtnData, error)
	DisposeCommand(ctx context.Context, data CommandDisposeData) error
	RouteAnnounceCommand(ctx context.Context) error   // (special) announces a new route to the main router
	RouteUnannounceCommand(ctx context.Context) error // (special) unannounces a route to the main router

	MessageCommand(ctx context.Context, data CommandMessageData) error
	GetMetaCommand(ctx context.Context, data CommandGetMetaData) (waveobj.MetaMapType, error)
	SetMetaCommand(ctx context.Context, data CommandSetMetaData) error
	SetViewCommand(ctx context.Context, data CommandBlockSetViewData) error
	ControllerInputCommand(ctx context.Context, data CommandBlockInputData) error
	ControllerStopCommand(ctx context.Context, blockId string) error
	ControllerResyncCommand(ctx context.Context, data CommandControllerResyncData) error
	ControllerAppendOutputCommand(ctx context.Context, data CommandControllerAppendOutputData) error
	ResolveIdsCommand(ctx context.Context, data CommandResolveIdsData) (CommandResolveIdsRtnData, error)
	CreateBlockCommand(ctx context.Context, data CommandCreateBlockData) (waveobj.ORef, error)
	CreateSubBlockCommand(ctx context.Context, data CommandCreateSubBlockData) (waveobj.ORef, error)
	DeleteBlockCommand(ctx context.Context, data CommandDeleteBlockData) error
	DeleteSubBlockCommand(ctx context.Context, data CommandDeleteBlockData) error
	WaitForRouteCommand(ctx context.Context, data CommandWaitForRouteData) (bool, error)

	FileMkdirCommand(ctx context.Context, data FileData) error
	FileCreateCommand(ctx context.Context, data FileData) error
	FileDeleteCommand(ctx context.Context, data CommandDeleteFileData) error
	FileAppendCommand(ctx context.Context, data FileData) error
	FileAppendIJsonCommand(ctx context.Context, data CommandAppendIJsonData) error
	FileWriteCommand(ctx context.Context, data FileData) error
	FileReadCommand(ctx context.Context, data FileData) (*FileData, error)
	FileReadStreamCommand(ctx context.Context, data FileData) <-chan RespOrErrorUnion[FileData]
	FileStreamTarCommand(ctx context.Context, data CommandRemoteStreamTarData) <-chan RespOrErrorUnion[iochantypes.Packet]
	FileMoveCommand(ctx context.Context, data CommandFileCopyData) error
	FileCopyCommand(ctx context.Context, data CommandFileCopyData) error
	FileInfoCommand(ctx context.Context, data FileData) (*FileInfo, error)
	FileListCommand(ctx context.Context, data FileListData) ([]*FileInfo, error)
	FileJoinCommand(ctx context.Context, paths []string) (*FileInfo, error)
	FileListStreamCommand(ctx context.Context, data FileListData) <-chan RespOrErrorUnion[CommandRemoteListEntriesRtnData]

	FileShareCapabilityCommand(ctx context.Context, path string) (FileShareCapability, error)
	EventPublishCommand(ctx context.Context, data wps.WaveEvent) error
	EventSubCommand(ctx context.Context, data wps.SubscriptionRequest) error
	EventUnsubCommand(ctx context.Context, data string) error
	EventUnsubAllCommand(ctx context.Context) error
	EventReadHistoryCommand(ctx context.Context, data CommandEventReadHistoryData) ([]*wps.WaveEvent, error)
	StreamTestCommand(ctx context.Context) chan RespOrErrorUnion[int]
	StreamWaveAiCommand(ctx context.Context, request WaveAIStreamRequest) chan RespOrErrorUnion[WaveAIPacketType]
	StreamCpuDataCommand(ctx context.Context, request CpuDataRequest) chan RespOrErrorUnion[TimeSeriesData]
	TestCommand(ctx context.Context, data string) error
	SetConfigCommand(ctx context.Context, data MetaSettingsType) error
	SetConnectionsConfigCommand(ctx context.Context, data ConnConfigRequest) error
	SetWorkspaceWidgetConfigCommand(ctx context.Context, data WorkspaceWidgetConfigRequest) error
	EnsureWorkspaceWidgetConfigCommand(ctx context.Context, data EnsureWorkspaceWidgetConfigRequest) error
	GetFullConfigCommand(ctx context.Context) (wconfig.FullConfigType, error)
	BlockInfoCommand(ctx context.Context, blockId string) (*BlockInfoData, error)
	WaveInfoCommand(ctx context.Context) (*WaveInfoData, error)
	WshActivityCommand(ct context.Context, data map[string]int) error
	ActivityCommand(ctx context.Context, data ActivityUpdate) error
	RecordTEventCommand(ctx context.Context, data telemetrydata.TEvent) error
	GetVarCommand(ctx context.Context, data CommandVarData) (*CommandVarResponseData, error)
	SetVarCommand(ctx context.Context, data CommandVarData) error
	PathCommand(ctx context.Context, data PathCommandData) (string, error)
	SendTelemetryCommand(ctx context.Context) error
	FetchSuggestionsCommand(ctx context.Context, data FetchSuggestionsData) (*FetchSuggestionsResponse, error)
	DisposeSuggestionsCommand(ctx context.Context, widgetId string) error
	GetTabCommand(ctx context.Context, tabId string) (*waveobj.Tab, error)

	// connection functions
	ConnStatusCommand(ctx context.Context) ([]ConnStatus, error)
	WslStatusCommand(ctx context.Context) ([]ConnStatus, error)
	ConnEnsureCommand(ctx context.Context, data ConnExtData) error
	ConnReinstallWshCommand(ctx context.Context, data ConnExtData) error
	ConnConnectCommand(ctx context.Context, connRequest ConnRequest) error
	ConnDisconnectCommand(ctx context.Context, connName string) error
	ConnListCommand(ctx context.Context) ([]string, error)
	ConnListAWSCommand(ctx context.Context) ([]string, error)
	WslListCommand(ctx context.Context) ([]string, error)
	WslDefaultDistroCommand(ctx context.Context) (string, error)
	DismissWshFailCommand(ctx context.Context, connName string) error
	ConnUpdateWshCommand(ctx context.Context, remoteInfo RemoteInfo) (bool, error)

	// eventrecv is special, it's handled internally by WshRpc with EventListener
	EventRecvCommand(ctx context.Context, data wps.WaveEvent) error

	// remotes
	RemoteStreamFileCommand(ctx context.Context, data CommandRemoteStreamFileData) chan RespOrErrorUnion[FileData]
	RemoteTarStreamCommand(ctx context.Context, data CommandRemoteStreamTarData) <-chan RespOrErrorUnion[iochantypes.Packet]
	RemoteFileCopyCommand(ctx context.Context, data CommandFileCopyData) (bool, error)
	RemoteListEntriesCommand(ctx context.Context, data CommandRemoteListEntriesData) chan RespOrErrorUnion[CommandRemoteListEntriesRtnData]
	RemoteFileInfoCommand(ctx context.Context, path string) (*FileInfo, error)
	RemoteFileTouchCommand(ctx context.Context, path string) error
	RemoteFileMoveCommand(ctx context.Context, data CommandFileCopyData) error
	RemoteFileDeleteCommand(ctx context.Context, data CommandDeleteFileData) error
	RemoteWriteFileCommand(ctx context.Context, data FileData) error
	RemoteFileJoinCommand(ctx context.Context, paths []string) (*FileInfo, error)
	RemoteMkdirCommand(ctx context.Context, path string) error
	RemoteStreamCpuDataCommand(ctx context.Context) chan RespOrErrorUnion[TimeSeriesData]
	RemoteGetInfoCommand(ctx context.Context) (RemoteInfo, error)
	RemoteInstallRcFilesCommand(ctx context.Context) error

	// emain
	WebSelectorCommand(ctx context.Context, data CommandWebSelectorData) ([]string, error)
	NotifyCommand(ctx context.Context, notificationOptions WaveNotificationOptions) error
	FocusWindowCommand(ctx context.Context, windowId string) error

	WorkspaceListCommand(ctx context.Context) ([]WorkspaceInfoData, error)
	GetUpdateChannelCommand(ctx context.Context) (string, error)

	// terminal
	VDomCreateContextCommand(ctx context.Context, data vdom.VDomCreateContext) (*waveobj.ORef, error)
	VDomAsyncInitiationCommand(ctx context.Context, data vdom.VDomAsyncInitiationRequest) error

	// ai
	AiSendMessageCommand(ctx context.Context, data AiMessageData) error

	// proc
	VDomRenderCommand(ctx context.Context, data vdom.VDomFrontendUpdate) chan RespOrErrorUnion[*vdom.VDomBackendUpdate]
	VDomUrlRequestCommand(ctx context.Context, data VDomUrlRequestData) chan RespOrErrorUnion[VDomUrlRequestResponse]
}

// for frontend
type WshServerCommandMeta struct {
	CommandType string `json:"commandtype"`
}

type RpcOpts struct {
	Timeout    int64  `json:"timeout,omitempty"`
	NoResponse bool   `json:"noresponse,omitempty"`
	Route      string `json:"route,omitempty"`

	StreamCancelFn func() `json:"-"` // this is an *output* parameter, set by the handler
}

const (
	ClientType_ConnServer      = "connserver"
	ClientType_BlockController = "blockcontroller"
)

type RpcContext struct {
	ClientType string `json:"ctype,omitempty"`
	BlockId    string `json:"blockid,omitempty"`
	TabId      string `json:"tabid,omitempty"`
	Conn       string `json:"conn,omitempty"`
}

func HackRpcContextIntoData(dataPtr any, rpcContext RpcContext) {
	dataVal := reflect.ValueOf(dataPtr).Elem()
	if dataVal.Kind() != reflect.Struct {
		return
	}
	dataType := dataVal.Type()
	for i := 0; i < dataVal.NumField(); i++ {
		field := dataVal.Field(i)
		if !field.IsZero() {
			continue
		}
		fieldType := dataType.Field(i)
		tag := fieldType.Tag.Get("wshcontext")
		if tag == "" {
			continue
		}
		switch tag {
		case "BlockId":
			field.SetString(rpcContext.BlockId)
		case "TabId":
			field.SetString(rpcContext.TabId)
		case "BlockORef":
			if rpcContext.BlockId != "" {
				field.Set(reflect.ValueOf(waveobj.MakeORef(waveobj.OType_Block, rpcContext.BlockId)))
			}
		default:
			log.Printf("invalid wshcontext tag: %q in type(%T)", tag, dataPtr)
		}
	}
}

type CommandAuthenticateRtnData struct {
	RouteId   string `json:"routeid"`
	AuthToken string `json:"authtoken,omitempty"`

	// these fields are only set when doing a token swap
	Env            map[string]string `json:"env,omitempty"`
	InitScriptText string            `json:"initscripttext,omitempty"`
}

type CommandAuthenticateTokenData struct {
	Token string `json:"token"`
}

type CommandDisposeData struct {
	RouteId string `json:"routeid"`
	// auth token travels in the packet directly
}

type CommandMessageData struct {
	ORef    waveobj.ORef `json:"oref" wshcontext:"BlockORef"`
	Message string       `json:"message"`
}

type CommandGetMetaData struct {
	ORef waveobj.ORef `json:"oref" wshcontext:"BlockORef"`
}

type CommandSetMetaData struct {
	ORef waveobj.ORef        `json:"oref" wshcontext:"BlockORef"`
	Meta waveobj.MetaMapType `json:"meta"`
}

type CommandResolveIdsData struct {
	BlockId string   `json:"blockid" wshcontext:"BlockId"`
	Ids     []string `json:"ids"`
}

type CommandResolveIdsRtnData struct {
	ResolvedIds map[string]waveobj.ORef `json:"resolvedids"`
}

type CommandCreateBlockData struct {
	TabId         string               `json:"tabid" wshcontext:"TabId"`
	BlockDef      *waveobj.BlockDef    `json:"blockdef"`
	RtOpts        *waveobj.RuntimeOpts `json:"rtopts,omitempty"`
	Magnified     bool                 `json:"magnified,omitempty"`
	Ephemeral     bool                 `json:"ephemeral,omitempty"`
	TargetBlockId string               `json:"targetblockid,omitempty"`
	TargetAction  string               `json:"targetaction,omitempty"` // "replace", "splitright", "splitdown", "splitleft", "splitup"
}

type CommandCreateSubBlockData struct {
	ParentBlockId string            `json:"parentblockid"`
	BlockDef      *waveobj.BlockDef `json:"blockdef"`
}

type CommandBlockSetViewData struct {
	BlockId string `json:"blockid" wshcontext:"BlockId"`
	View    string `json:"view"`
}

type CommandControllerResyncData struct {
	ForceRestart bool                 `json:"forcerestart,omitempty"`
	TabId        string               `json:"tabid" wshcontext:"TabId"`
	BlockId      string               `json:"blockid" wshcontext:"BlockId"`
	RtOpts       *waveobj.RuntimeOpts `json:"rtopts,omitempty"`
}

type CommandControllerAppendOutputData struct {
	BlockId string `json:"blockid"`
	Data64  string `json:"data64"`
}

type CommandBlockInputData struct {
	BlockId     string            `json:"blockid" wshcontext:"BlockId"`
	InputData64 string            `json:"inputdata64,omitempty"`
	SigName     string            `json:"signame,omitempty"`
	TermSize    *waveobj.TermSize `json:"termsize,omitempty"`
}

type FileDataAt struct {
	Offset int64 `json:"offset"`
	Size   int   `json:"size,omitempty"`
}

type FileData struct {
	Info    *FileInfo   `json:"info,omitempty"`
	Data64  string      `json:"data64,omitempty"`
	Entries []*FileInfo `json:"entries,omitempty"`
	At      *FileDataAt `json:"at,omitempty"` // if set, this turns read/write ops to ReadAt/WriteAt ops (len is only used for ReadAt)
}

type FileInfo struct {
	Path          string      `json:"path"`          // cleaned path (may have "~")
	Dir           string      `json:"dir,omitempty"` // returns the directory part of the path (if this is a a directory, it will be equal to Path).  "~" will be expanded, and separators will be normalized to "/"
	Name          string      `json:"name,omitempty"`
	NotFound      bool        `json:"notfound,omitempty"`
	Opts          *FileOpts   `json:"opts,omitempty"`
	Size          int64       `json:"size,omitempty"`
	Meta          *FileMeta   `json:"meta,omitempty"`
	Mode          os.FileMode `json:"mode,omitempty"`
	ModeStr       string      `json:"modestr,omitempty"`
	ModTime       int64       `json:"modtime,omitempty"`
	IsDir         bool        `json:"isdir,omitempty"`
	SupportsMkdir bool        `json:"supportsmkdir,omitempty"`
	MimeType      string      `json:"mimetype,omitempty"`
	ReadOnly      bool        `json:"readonly,omitempty"` // this is not set for fileinfo's returned from directory listings
}

type FileOpts struct {
	MaxSize     int64 `json:"maxsize,omitempty"`
	Circular    bool  `json:"circular,omitempty"`
	IJson       bool  `json:"ijson,omitempty"`
	IJsonBudget int   `json:"ijsonbudget,omitempty"`
	Truncate    bool  `json:"truncate,omitempty"`
	Append      bool  `json:"append,omitempty"`
}

type FileMeta = map[string]any

type FileListStreamResponse <-chan RespOrErrorUnion[CommandRemoteListEntriesRtnData]

type FileListData struct {
	Path string        `json:"path"`
	Opts *FileListOpts `json:"opts,omitempty"`
}

type FileListOpts struct {
	All    bool `json:"all,omitempty"`
	Offset int  `json:"offset,omitempty"`
	Limit  int  `json:"limit,omitempty"`
}

type FileCreateData struct {
	Path string         `json:"path"`
	Meta map[string]any `json:"meta,omitempty"`
	Opts *FileOpts      `json:"opts,omitempty"`
}

type CommandAppendIJsonData struct {
	ZoneId   string        `json:"zoneid" wshcontext:"BlockId"`
	FileName string        `json:"filename"`
	Data     ijson.Command `json:"data"`
}

type CommandWaitForRouteData struct {
	RouteId string `json:"routeid"`
	WaitMs  int    `json:"waitms"`
}

type CommandDeleteBlockData struct {
	BlockId string `json:"blockid" wshcontext:"BlockId"`
}

type CommandEventReadHistoryData struct {
	Event    string `json:"event"`
	Scope    string `json:"scope"`
	MaxItems int    `json:"maxitems"`
}

type WaveAIStreamRequest struct {
	ClientId string                    `json:"clientid,omitempty"`
	Opts     *WaveAIOptsType           `json:"opts"`
	Prompt   []WaveAIPromptMessageType `json:"prompt"`
}

type WaveAIPromptMessageType struct {
	Role    string `json:"role"`
	Content string `json:"content"`
	Name    string `json:"name,omitempty"`
}

type WaveAIOptsType struct {
	Model      string `json:"model"`
	APIType    string `json:"apitype,omitempty"`
	APIToken   string `json:"apitoken"`
	OrgID      string `json:"orgid,omitempty"`
	APIVersion string `json:"apiversion,omitempty"`
	BaseURL    string `json:"baseurl,omitempty"`
	MaxTokens  int    `json:"maxtokens,omitempty"`
	MaxChoices int    `json:"maxchoices,omitempty"`
	TimeoutMs  int    `json:"timeoutms,omitempty"`
}

type WaveAIPacketType struct {
	Type         string           `json:"type"`
	Model        string           `json:"model,omitempty"`
	Created      int64            `json:"created,omitempty"`
	FinishReason string           `json:"finish_reason,omitempty"`
	Usage        *WaveAIUsageType `json:"usage,omitempty"`
	Index        int              `json:"index,omitempty"`
	Text         string           `json:"text,omitempty"`
	Error        string           `json:"error,omitempty"`
}

type WaveAIUsageType struct {
	PromptTokens     int `json:"prompt_tokens,omitempty"`
	CompletionTokens int `json:"completion_tokens,omitempty"`
	TotalTokens      int `json:"total_tokens,omitempty"`
}

type CpuDataRequest struct {
	Id    string `json:"id"`
	Count int    `json:"count"`
}

type CpuDataType struct {
	Time  int64   `json:"time"`
	Value float64 `json:"value"`
}

type CommandDeleteFileData struct {
	Path      string `json:"path"`
	Recursive bool   `json:"recursive"`
}

type CommandFileCopyData struct {
	SrcUri  string        `json:"srcuri"`
	DestUri string        `json:"desturi"`
	Opts    *FileCopyOpts `json:"opts,omitempty"`
}

type CommandRemoteStreamTarData struct {
	Path string        `json:"path"`
	Opts *FileCopyOpts `json:"opts,omitempty"`
}

type FileCopyOpts struct {
	Overwrite bool  `json:"overwrite,omitempty"`
	Recursive bool  `json:"recursive,omitempty"` // only used for move, always true for copy
	Merge     bool  `json:"merge,omitempty"`
	Timeout   int64 `json:"timeout,omitempty"`
}

type CommandRemoteStreamFileData struct {
	Path      string `json:"path"`
	ByteRange string `json:"byterange,omitempty"`
}

type CommandRemoteListEntriesData struct {
	Path string        `json:"path"`
	Opts *FileListOpts `json:"opts,omitempty"`
}

type CommandRemoteListEntriesRtnData struct {
	FileInfo []*FileInfo `json:"fileinfo,omitempty"`
}

type ConnRequest struct {
	Host       string               `json:"host"`
	Keywords   wconfig.ConnKeywords `json:"keywords,omitempty"`
	LogBlockId string               `json:"logblockid,omitempty"`
}

type RemoteInfo struct {
	ClientArch    string `json:"clientarch"`
	ClientOs      string `json:"clientos"`
	ClientVersion string `json:"clientversion"`
	Shell         string `json:"shell"`
}

const (
	TimeSeries_Cpu = "cpu"
)

type TimeSeriesData struct {
	Ts     int64              `json:"ts"`
	Values map[string]float64 `json:"values"`
}

type MetaSettingsType struct {
	waveobj.MetaMapType
}

func (m *MetaSettingsType) UnmarshalJSON(data []byte) error {
	var metaMap waveobj.MetaMapType
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := decoder.Decode(&metaMap); err != nil {
		return err
	}
	*m = MetaSettingsType{MetaMapType: metaMap}
	return nil
}

func (m MetaSettingsType) MarshalJSON() ([]byte, error) {
	return json.Marshal(m.MetaMapType)
}

type ConnConfigRequest struct {
	Host        string              `json:"host"`
	MetaMapType waveobj.MetaMapType `json:"metamaptype"`
}

type WorkspaceWidgetConfigRequest struct {
	WorkspaceId string                `json:"workspaceid" wshcontext:"WorkspaceId"`
	WidgetKey   string                `json:"widgetkey"`
	Config      wconfig.WidgetConfigType `json:"config"`
}

type EnsureWorkspaceWidgetConfigRequest struct {
	WorkspaceId string `json:"workspaceid" wshcontext:"WorkspaceId"`
}

type ConnStatus struct {
	Status        string `json:"status"`
	WshEnabled    bool   `json:"wshenabled"`
	Connection    string `json:"connection"`
	Connected     bool   `json:"connected"`
	HasConnected  bool   `json:"hasconnected"` // true if it has *ever* connected successfully
	ActiveConnNum int    `json:"activeconnnum"`
	Error         string `json:"error,omitempty"`
	WshError      string `json:"wsherror,omitempty"`
	NoWshReason   string `json:"nowshreason,omitempty"`
	WshVersion    string `json:"wshversion,omitempty"`
}

type WebSelectorOpts struct {
	All   bool `json:"all,omitempty"`
	Inner bool `json:"inner,omitempty"`
}

type CommandWebSelectorData struct {
	WorkspaceId string           `json:"workspaceid"`
	BlockId     string           `json:"blockid" wshcontext:"BlockId"`
	TabId       string           `json:"tabid" wshcontext:"TabId"`
	Selector    string           `json:"selector"`
	Opts        *WebSelectorOpts `json:"opts,omitempty"`
}

type BlockInfoData struct {
	BlockId     string         `json:"blockid"`
	TabId       string         `json:"tabid"`
	WorkspaceId string         `json:"workspaceid"`
	Block       *waveobj.Block `json:"block"`
	Files       []*FileInfo    `json:"files"`
}

type WaveNotificationOptions struct {
	Title  string `json:"title,omitempty"`
	Body   string `json:"body,omitempty"`
	Silent bool   `json:"silent,omitempty"`
}

type VDomUrlRequestData struct {
	Method  string            `json:"method"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
	Body    []byte            `json:"body,omitempty"`
}

type VDomUrlRequestResponse struct {
	StatusCode int               `json:"statuscode,omitempty"`
	Headers    map[string]string `json:"headers,omitempty"`
	Body       []byte            `json:"body,omitempty"`
}

type WaveInfoData struct {
	Version   string `json:"version"`
	ClientId  string `json:"clientid"`
	BuildTime string `json:"buildtime"`
	ConfigDir string `json:"configdir"`
	DataDir   string `json:"datadir"`
}

type WorkspaceInfoData struct {
	WindowId      string             `json:"windowid"`
	WorkspaceData *waveobj.Workspace `json:"workspacedata"`
}

type AiMessageData struct {
	Message string `json:"message,omitempty"`
}

type CommandVarData struct {
	Key      string `json:"key"`
	Val      string `json:"val,omitempty"`
	Remove   bool   `json:"remove,omitempty"`
	ZoneId   string `json:"zoneid"`
	FileName string `json:"filename"`
}

type CommandVarResponseData struct {
	Key    string `json:"key"`
	Val    string `json:"val"`
	Exists bool   `json:"exists"`
}

type PathCommandData struct {
	PathType     string `json:"pathtype"`
	Open         bool   `json:"open"`
	OpenExternal bool   `json:"openexternal"`
	TabId        string `json:"tabid" wshcontext:"TabId"`
}

type ActivityDisplayType struct {
	Width    int     `json:"width"`
	Height   int     `json:"height"`
	DPR      float64 `json:"dpr"`
	Internal bool    `json:"internal,omitempty"`
}

type ActivityUpdate struct {
	FgMinutes     int                   `json:"fgminutes,omitempty"`
	ActiveMinutes int                   `json:"activeminutes,omitempty"`
	OpenMinutes   int                   `json:"openminutes,omitempty"`
	NumTabs       int                   `json:"numtabs,omitempty"`
	NewTab        int                   `json:"newtab,omitempty"`
	NumBlocks     int                   `json:"numblocks,omitempty"`
	NumWindows    int                   `json:"numwindows,omitempty"`
	NumWS         int                   `json:"numws,omitempty"`
	NumWSNamed    int                   `json:"numwsnamed,omitempty"`
	NumSSHConn    int                   `json:"numsshconn,omitempty"`
	NumWSLConn    int                   `json:"numwslconn,omitempty"`
	NumMagnify    int                   `json:"nummagnify,omitempty"`
	NumPanics     int                   `json:"numpanics,omitempty"`
	NumAIReqs     int                   `json:"numaireqs,omitempty"`
	Startup       int                   `json:"startup,omitempty"`
	Shutdown      int                   `json:"shutdown,omitempty"`
	SetTabTheme   int                   `json:"settabtheme,omitempty"`
	BuildTime     string                `json:"buildtime,omitempty"`
	Displays      []ActivityDisplayType `json:"displays,omitempty"`
	Renderers     map[string]int        `json:"renderers,omitempty"`
	Blocks        map[string]int        `json:"blocks,omitempty"`
	WshCmds       map[string]int        `json:"wshcmds,omitempty"`
	Conn          map[string]int        `json:"conn,omitempty"`
}

type ConnExtData struct {
	ConnName   string `json:"connname"`
	LogBlockId string `json:"logblockid,omitempty"`
}

type FetchSuggestionsData struct {
	SuggestionType string `json:"suggestiontype"`
	Query          string `json:"query"`
	WidgetId       string `json:"widgetid"`
	ReqNum         int    `json:"reqnum"`
	FileCwd        string `json:"file:cwd,omitempty"`
	FileDirOnly    bool   `json:"file:dironly,omitempty"`
	FileConnection string `json:"file:connection,omitempty"`
}

type FetchSuggestionsResponse struct {
	ReqNum      int              `json:"reqnum"`
	Suggestions []SuggestionType `json:"suggestions"`
}

type SuggestionType struct {
	Type         string `json:"type"`
	SuggestionId string `json:"suggestionid"`
	Display      string `json:"display"`
	SubText      string `json:"subtext,omitempty"`
	Icon         string `json:"icon,omitempty"`
	IconColor    string `json:"iconcolor,omitempty"`
	IconSrc      string `json:"iconsrc,omitempty"`
	MatchPos     []int  `json:"matchpos,omitempty"`
	SubMatchPos  []int  `json:"submatchpos,omitempty"`
	Score        int    `json:"score,omitempty"`
	FileMimeType string `json:"file:mimetype,omitempty"`
	FilePath     string `json:"file:path,omitempty"`
	FileName     string `json:"file:name,omitempty"`
	UrlUrl       string `json:"url:url,omitempty"`
}

// FileShareCapability represents the capabilities of a file share
type FileShareCapability struct {
	// CanAppend indicates whether the file share supports appending to files
	CanAppend bool `json:"canappend"`
	// CanMkdir indicates whether the file share supports creating directories
	CanMkdir bool `json:"canmkdir"`
}
