const { Client } = require('@modelcontextprotocol/sdk/client/index.js');
const { StdioClientTransport } = require('@modelcontextprotocol/sdk/client/stdio.js');

class MCPBridge {
  constructor() {
    this.client = null;
    this.isConnected = false;
    this.terminalHandlers = new Map(); // blockId -> handler functions
  }

  async connect(command = 'claude') {
    try {
      // Use full path if available
      const claudePath = process.env.CLAUDE_PATH || '~/.claude/local/claude';
      const transport = new StdioClientTransport({
        command: claudePath,
        args: [],
        env: { ...process.env }
      });

      this.client = new Client({
        name: 'wave-terminal-mcp-bridge',
        version: '1.0.0'
      }, {
        capabilities: {}
      });

      await this.client.connect(transport);
      this.isConnected = true;
      console.log('✅ MCP Bridge connected to Claude CLI');
      return true;
    } catch (error) {
      console.error('❌ Failed to connect MCP Bridge:', error);
      this.isConnected = false;
      return false;
    }
  }

  // Enhanced terminal input with proper handling for interactive CLIs
  async sendTerminalInput(blockId, inputData, options = {}) {
    if (!blockId) {
      throw new Error('blockId is required');
    }
    
    const {
      waitForPrompt = false,
      promptTimeout = 3000,
      inputType = 'text'
    } = options;

    try {
      // Handle different input types
      let processedInput = inputData;
      
      if (inputType === 'text') {
        // For empty input (Enter key), use \r
        if (!inputData || inputData === '') {
          processedInput = '\r';
        }
        // For commands, ensure they end with \r
        else if (!inputData.endsWith('\r') && !inputData.endsWith('\n')) {
          processedInput = inputData + '\r';
        }
        // Replace any standalone \n with \r
        else {
          processedInput = inputData.replace(/\n/g, '\r');
        }
      }

      // Check if this is an interactive CLI command
      const isInteractive = this.isInteractiveCLI(inputData);
      
      if (isInteractive && waitForPrompt) {
        // For interactive CLIs, wait for prompt before sending
        const promptReady = await this.waitForPrompt(blockId, promptTimeout);
        if (!promptReady) {
          console.warn('⚠️ Prompt not detected, sending anyway');
        }
      }

      // Send the processed input
      const result = await this.callTool('send_terminal_input', {
        block_id: blockId,
        input_data: processedInput,
        input_type: inputType
      });

      // Log for debugging
      console.log(`📤 Sent to terminal (${blockId}): ${JSON.stringify(processedInput)}`);
      
      return result;
    } catch (error) {
      console.error('❌ Failed to send terminal input:', error);
      throw error;
    }
  }

  // Execute command with expect-like behavior
  async executeCommand(blockId, command, options = {}) {
    const {
      waitForCompletion = true,
      timeout = 10000,
      waitForPromptFirst = true
    } = options;

    try {
      // First, wait for a prompt if requested
      if (waitForPromptFirst) {
        await this.waitForPrompt(blockId, 3000);
      }

      // Send the command with proper line ending
      const commandToSend = command.endsWith('\r') ? command : command + '\r';
      
      const result = await this.callTool('execute_command', {
        block_id: blockId,
        command: command, // Tool might handle line endings internally
        timeout: timeout
      });

      console.log(`✅ Command executed: ${command}`);
      return result;
    } catch (error) {
      console.error('❌ Failed to execute command:', error);
      throw error;
    }
  }

  // Wait for prompt pattern in terminal output
  async waitForPrompt(blockId, timeout = 3000) {
    const startTime = Date.now();
    const checkInterval = 100;
    
    // Common prompt patterns
    const promptPatterns = [
      />[\s]*$/,           // Claude CLI and generic prompts
      /\$[\s]*$/,          // Shell prompt
      /#[\s]*$/,           // Root prompt
      /claude>[\s]*$/,     // Claude-specific prompt
      />>>[\s]*$/,         // Python prompt
      />>[\s]*$/,          // Some REPL prompts
    ];

    while (Date.now() - startTime < timeout) {
      try {
        // Get terminal content
        const content = await this.getBlockContent(blockId);
        
        // Clean ANSI sequences for matching
        const cleanContent = this.stripANSI(content);
        
        // Check for any prompt pattern
        for (const pattern of promptPatterns) {
          if (pattern.test(cleanContent)) {
            console.log('✅ Prompt detected');
            return true;
          }
        }
        
        // Wait before next check
        await new Promise(resolve => setTimeout(resolve, checkInterval));
      } catch (error) {
        console.warn('⚠️ Error checking for prompt:', error);
        break;
      }
    }
    
    console.warn('⚠️ Prompt not detected within timeout');
    return false;
  }

  // Get terminal content for analysis
  async getBlockContent(blockId) {
    try {
      const result = await this.callTool('get_block_content', {
        block_id: blockId
      });
      return result?.content || '';
    } catch (error) {
      console.error('❌ Failed to get block content:', error);
      return '';
    }
  }

  // Check if command is an interactive CLI
  isInteractiveCLI(command) {
    const interactiveCLIs = [
      'claude',
      'python', 'python3',
      'node',
      'irb', 'pry',
      'ghci',
      'sqlite3',
      'mysql',
      'psql',
      'redis-cli',
      'mongo',
      'npm', 'yarn', 'pnpm'
    ];

    const baseCommand = command.split(/\s+/)[0];
    const cmdName = baseCommand.split('/').pop();
    
    return interactiveCLIs.some(cli => cmdName.startsWith(cli));
  }

  // Strip ANSI escape sequences
  stripANSI(str) {
    return str.replace(/\x1b\[[0-9;]*[a-zA-Z]/g, '')
              .replace(/\x1b\].*?\x07/g, '')
              .replace(/\x1b[PX^_].*?\x1b\\/g, '')
              .replace(/\x1b\[[\?!][0-9;]*[a-zA-Z]/g, '');
  }

  // Call MCP tool
  async callTool(toolName, args) {
    if (!this.isConnected) {
      console.log('🔄 Attempting to reconnect MCP Bridge...');
      await this.connect();
      if (!this.isConnected) {
        throw new Error('MCP Bridge not connected');
      }
    }
    
    try {
      const result = await this.client.callTool({
        name: `mcp__wave-terminal__${toolName}`,
        arguments: args
      });
      return result;
    } catch (error) {
      console.error(`❌ Tool call failed (${toolName}):`, error);
      throw error;
    }
  }

  // Test the bridge with Claude CLI
  async testClaudeCLI(blockId) {
    console.log('🧪 Testing Claude CLI interaction...');
    
    try {
      // Wait for initial prompt
      console.log('⏳ Waiting for Claude CLI prompt...');
      const hasPrompt = await this.waitForPrompt(blockId, 5000);
      
      if (!hasPrompt) {
        console.warn('⚠️ No prompt detected, trying anyway...');
      }
      
      // Send /doctor command
      console.log('📤 Sending /doctor command...');
      await this.sendTerminalInput(blockId, '/doctor', {
        waitForPrompt: true
      });
      
      // Wait a bit for response
      await new Promise(resolve => setTimeout(resolve, 2000));
      
      // Get output
      const output = await this.getBlockContent(blockId);
      console.log('📥 Terminal output:', output.slice(-500)); // Last 500 chars
      
      return true;
    } catch (error) {
      console.error('❌ Test failed:', error);
      return false;
    }
  }
}

// Export for use in other modules
module.exports = { MCPBridge };

// Example usage
if (require.main === module) {
  (async () => {
    const bridge = new MCPBridge();
    
    // Connect to Claude CLI
    await bridge.connect();
    
    // Example: Test with a specific block ID
    const blockId = process.argv[2];
    if (blockId) {
      await bridge.testClaudeCLI(blockId);
    } else {
      console.log('Usage: node mcp-bridge-enhanced.cjs <block-id>');
    }
  })();
}