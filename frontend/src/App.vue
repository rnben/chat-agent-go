<template>
  <div class="app">
    <!-- 侧边栏：会话列表 -->
    <aside class="sidebar">
      <div class="sidebar-header">
        <h2>📦 订单助手</h2>
        <button class="btn-new" @click="createSession">+ 新对话</button>
      </div>
      <div class="session-list">
        <div
          v-for="session in sessions"
          :key="session.id"
          :class="['session-item', { active: currentSessionId === session.id }]"
          @click="selectSession(session.id)"
        >
          <span class="session-title">{{ session.title || '新会话' }}</span>
          <button class="btn-delete" @click.stop="deleteSession(session.id)">×</button>
        </div>
        <div v-if="sessions.length === 0" class="empty-hint">
          暂无会话
        </div>
      </div>
    </aside>

    <!-- 主聊天区域 -->
    <main class="chat-main">
      <div class="chat-header">
        <span>{{ currentSessionTitle }}</span>
      </div>
      
      <div class="messages" ref="messagesContainer">
        <div v-if="messages.length === 0" class="welcome">
          <p>👋 你好！我是订单查询助手</p>
          <p>你可以问我：</p>
          <ul>
            <li>查询订单 <code>ORD-20260515-001</code> 的状态</li>
            <li>查看用户 <code>user_001</code> 的所有订单</li>
          </ul>
        </div>
        
        <div
          v-for="(msg, index) in messages"
          :key="index"
          :class="['message', msg.role]"
        >
          <div class="message-avatar">
            {{ msg.role === 'user' ? '👤' : msg.role === 'tool' ? '🔧' : '🤖' }}
          </div>
          <div class="message-content">
            <div class="message-text" v-html="formatContent(msg.content)"></div>
            <div v-if="msg.tool_call" class="tool-call">
              🔧 调用工具: {{ msg.tool_call.name }}
            </div>
            <div v-if="msg.tool_result" class="tool-result">
              <div class="tool-result-header">📋 查询结果</div>
              <pre>{{ msg.tool_result }}</pre>
            </div>
          </div>
        </div>
        
        <div v-if="isLoading" class="message assistant loading">
          <div class="message-avatar">🤖</div>
          <div class="message-content">
            <span class="typing">思考中<span class="dot">.</span><span class="dot">.</span><span class="dot">.</span></span>
          </div>
        </div>
      </div>

      <div class="input-area">
        <textarea
          v-model="userInput"
          @keydown.enter.exact.prevent="sendMessage"
          placeholder="输入消息... (Enter 发送)"
          :disabled="isLoading"
          rows="1"
        ></textarea>
        <button 
          class="btn-send" 
          @click="sendMessage" 
          :disabled="isLoading || !userInput.trim()"
        >
          {{ isLoading ? '发送中...' : '发送' }}
        </button>
        <button 
          v-if="isLoading" 
          class="btn-stop" 
          @click="stopGeneration"
        >
          ⏹ 停止
        </button>
      </div>
    </main>
  </div>
</template>

<script>
import { ref, onMounted, nextTick, watch } from 'vue'

export default {
  name: 'App',
  setup() {
    // 状态
    const sessions = ref([])
    const currentSessionId = ref('')
    const messages = ref([])
    const userInput = ref('')
    const isLoading = ref(false)
    const messagesContainer = ref(null)
    
    // SSE 控制器
    let currentAbortController = null

    // API 基础地址
    const API_BASE = '/api'

    // 加载会话列表
    const loadSessions = async () => {
      try {
        const res = await fetch(`${API_BASE}/sessions`)
        sessions.value = await res.json()
      } catch (e) {
        console.error('加载会话失败:', e)
      }
    }

    // 创建新会话
    const createSession = async () => {
      try {
        const res = await fetch(`${API_BASE}/sessions`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' }
        })
        const session = await res.json()
        sessions.value.unshift(session)
        selectSession(session.id)
      } catch (e) {
        console.error('创建会话失败:', e)
      }
    }

    // 选择会话
    const selectSession = async (sessionId) => {
      currentSessionId.value = sessionId
      await loadHistory(sessionId)
    }

    // 加载历史
    const loadHistory = async (sessionId) => {
      try {
        const res = await fetch(`${API_BASE}/sessions/${sessionId}/history`)
        messages.value = await res.json()
        await nextTick()
        scrollToBottom()
      } catch (e) {
        console.error('加载历史失败:', e)
      }
    }

    // 删除会话
    const deleteSession = async (sessionId) => {
      try {
        await fetch(`${API_BASE}/sessions/${sessionId}`, { method: 'DELETE' })
        sessions.value = sessions.value.filter(s => s.id !== sessionId)
        if (currentSessionId.value === sessionId) {
          currentSessionId.value = ''
          messages.value = []
        }
      } catch (e) {
        console.error('删除会话失败:', e)
      }
    }

    // 发送消息
    const sendMessage = async () => {
      const text = userInput.value.trim()
      if (!text || isLoading.value) return

      // 如果没有会话，先创建
      if (!currentSessionId.value) {
        await createSession()
      }

      // 添加用户消息到界面
      messages.value.push({
        role: 'user',
        content: text
      })
      
      userInput.value = ''
      isLoading.value = true
      await nextTick()
      scrollToBottom()

      // 用于累积助手回复
      let assistantContent = ''
      let currentToolCall = null
      let currentToolResult = null

      // 创建中止控制器
      currentAbortController = new AbortController()

      try {
        const res = await fetch(`${API_BASE}/chat`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            session_id: currentSessionId.value,
            message: text
          }),
          signal: currentAbortController.signal
        })

        if (!res.ok) {
          throw new Error(`HTTP ${res.status}`)
        }

        const reader = res.body.getReader()
        const decoder = new TextDecoder()
        let buffer = ''

        while (true) {
          const { done, value } = await reader.read()
          if (done) break

          buffer += decoder.decode(value, { stream: true })
          const lines = buffer.split('\n')
          buffer = lines.pop() || ''

          let eventType = ''
          let data = ''

          for (const line of lines) {
            if (line.startsWith('event: ')) {
              eventType = line.slice(7)
            } else if (line.startsWith('data: ')) {
              data = line.slice(6)
            } else if (line === '' && eventType && data) {
              // 处理事件
              try {
                const parsed = JSON.parse(data)
                
                if (eventType === 'session') {
                  currentSessionId.value = parsed.session_id
                  loadSessions()
                } else if (eventType === 'content') {
                  assistantContent += parsed.content
                  updateAssistantMessage(assistantContent)
                } else if (eventType === 'tool_call') {
                  currentToolCall = parsed
                  assistantContent += '\n\n🔧 正在查询...'
                  updateAssistantMessage(assistantContent)
                } else if (eventType === 'tool_result') {
                  assistantContent = assistantContent.replace('\n\n🔧 正在查询...', '')
                  currentToolResult = parsed.result
                  updateAssistantMessage(assistantContent, currentToolResult)
                } else if (eventType === 'done') {
                  // 完成
                } else if (eventType === 'error') {
                  assistantContent += `\n\n❌ 错误: ${parsed.error}`
                  updateAssistantMessage(assistantContent)
                }
              } catch (e) {
                console.error('解析事件失败:', e)
              }
              
              eventType = ''
              data = ''
            }
          }
        }

        // 刷新会话列表
        loadSessions()

      } catch (e) {
        if (e.name === 'AbortError') {
          assistantContent += '\n\n⏹ 已停止生成'
        } else {
          assistantContent += `\n\n❌ 请求失败: ${e.message}`
        }
        updateAssistantMessage(assistantContent)
      } finally {
        isLoading.value = false
        currentAbortController = null
      }
    }

    // 更新助手消息
    const updateAssistantMessage = (content, toolResult = null) => {
      const lastMsg = messages.value[messages.value.length - 1]
      if (lastMsg && lastMsg.role === 'assistant') {
        lastMsg.content = content
        if (toolResult) {
          lastMsg.tool_result = toolResult
        }
      } else {
        messages.value.push({
          role: 'assistant',
          content: content,
          tool_result: toolResult
        })
      }
      nextTick(() => scrollToBottom())
    }

    // 停止生成
    const stopGeneration = () => {
      if (currentAbortController) {
        currentAbortController.abort()
      }
    }

    // 滚动到底部
    const scrollToBottom = () => {
      if (messagesContainer.value) {
        messagesContainer.value.scrollTop = messagesContainer.value.scrollHeight
      }
    }

    // 格式化内容（简单的 markdown 渲染）
    const formatContent = (content) => {
      if (!content) return ''
      return content
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/\n/g, '<br>')
        .replace(/`([^`]+)`/g, '<code>$1</code>')
        .replace(/\*\*([^*]+)\*\*/g, '<strong>$1</strong>')
    }

    // 计算属性：当前会话标题
    const currentSessionTitle = ref('订单查询助手')
    watch(currentSessionId, (id) => {
      const session = sessions.value.find(s => s.id === id)
      currentSessionTitle.value = session?.title || '订单查询助手'
    })

    // 初始化
    onMounted(() => {
      loadSessions()
    })

    return {
      sessions,
      currentSessionId,
      currentSessionTitle,
      messages,
      userInput,
      isLoading,
      messagesContainer,
      createSession,
      selectSession,
      deleteSession,
      sendMessage,
      stopGeneration,
      formatContent
    }
  }
}
</script>

<style>
* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

body {
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
  background: #f5f5f5;
  height: 100vh;
}

#app {
  height: 100vh;
}

.app {
  display: flex;
  height: 100vh;
}

/* 侧边栏 */
.sidebar {
  width: 280px;
  background: #2d3748;
  color: white;
  display: flex;
  flex-direction: column;
}

.sidebar-header {
  padding: 20px;
  border-bottom: 1px solid #4a5568;
}

.sidebar-header h2 {
  font-size: 18px;
  margin-bottom: 15px;
}

.btn-new {
  width: 100%;
  padding: 10px;
  background: #4299e1;
  color: white;
  border: none;
  border-radius: 8px;
  cursor: pointer;
  font-size: 14px;
}

.btn-new:hover {
  background: #3182ce;
}

.session-list {
  flex: 1;
  overflow-y: auto;
  padding: 10px;
}

.session-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px;
  margin-bottom: 5px;
  background: #4a5568;
  border-radius: 8px;
  cursor: pointer;
}

.session-item:hover {
  background: #5a6778;
}

.session-item.active {
  background: #4299e1;
}

.session-title {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 14px;
}

.btn-delete {
  background: none;
  border: none;
  color: #a0aec0;
  cursor: pointer;
  font-size: 18px;
  padding: 0 5px;
}

.btn-delete:hover {
  color: #e53e3e;
}

.empty-hint {
  text-align: center;
  color: #a0aec0;
  padding: 20px;
  font-size: 14px;
}

/* 主聊天区 */
.chat-main {
  flex: 1;
  display: flex;
  flex-direction: column;
  background: white;
}

.chat-header {
  padding: 15px 20px;
  border-bottom: 1px solid #e2e8f0;
  font-weight: 600;
  color: #2d3748;
}

.messages {
  flex: 1;
  overflow-y: auto;
  padding: 20px;
}

.welcome {
  text-align: center;
  color: #718096;
  padding: 40px;
}

.welcome p {
  margin-bottom: 10px;
}

.welcome ul {
  list-style: none;
  margin-top: 15px;
}

.welcome li {
  margin: 10px 0;
}

.welcome code {
  background: #edf2f7;
  padding: 2px 6px;
  border-radius: 4px;
  color: #2d3748;
}

.message {
  display: flex;
  margin-bottom: 20px;
  animation: fadeIn 0.3s;
}

@keyframes fadeIn {
  from { opacity: 0; transform: translateY(10px); }
  to { opacity: 1; transform: translateY(0); }
}

.message.user {
  flex-direction: row-reverse;
}

.message-avatar {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  background: #e2e8f0;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 20px;
  flex-shrink: 0;
}

.message.user .message-avatar {
  background: #4299e1;
}

.message-content {
  max-width: 70%;
  margin: 0 15px;
}

.message-text {
  padding: 12px 16px;
  border-radius: 16px;
  background: #e2e8f0;
  line-height: 1.6;
  word-break: break-word;
}

.message.user .message-text {
  background: #4299e1;
  color: white;
}

.message-text code {
  background: rgba(0,0,0,0.1);
  padding: 2px 6px;
  border-radius: 4px;
  font-family: monospace;
}

.message.assistant .message-text code {
  background: #edf2f7;
}

.tool-call {
  margin-top: 8px;
  padding: 8px 12px;
  background: #fef3c7;
  border-radius: 8px;
  font-size: 13px;
  color: #92400e;
}

.tool-result {
  margin-top: 10px;
  background: #f0fff4;
  border: 1px solid #9ae6b4;
  border-radius: 8px;
  overflow: hidden;
}

.tool-result-header {
  padding: 8px 12px;
  background: #9ae6b4;
  font-size: 12px;
  font-weight: 600;
  color: #22543d;
}

.tool-result pre {
  padding: 12px;
  font-size: 13px;
  white-space: pre-wrap;
  word-break: break-word;
  color: #2d3748;
  margin: 0;
}

/* 输入区 */
.input-area {
  padding: 15px 20px;
  border-top: 1px solid #e2e8f0;
  display: flex;
  gap: 10px;
}

.input-area textarea {
  flex: 1;
  padding: 12px 16px;
  border: 1px solid #e2e8f0;
  border-radius: 12px;
  resize: none;
  font-size: 15px;
  outline: none;
  font-family: inherit;
}

.input-area textarea:focus {
  border-color: #4299e1;
}

.btn-send {
  padding: 12px 24px;
  background: #4299e1;
  color: white;
  border: none;
  border-radius: 12px;
  cursor: pointer;
  font-size: 15px;
}

.btn-send:hover:not(:disabled) {
  background: #3182ce;
}

.btn-send:disabled {
  background: #a0aec0;
  cursor: not-allowed;
}

.btn-stop {
  padding: 12px 20px;
  background: #e53e3e;
  color: white;
  border: none;
  border-radius: 12px;
  cursor: pointer;
  font-size: 15px;
}

.btn-stop:hover {
  background: #c53030;
}

/* 加载动画 */
.typing .dot {
  animation: blink 1.4s infinite both;
}

.dot:nth-child(2) { animation-delay: 0.2s; }
.dot:nth-child(3) { animation-delay: 0.4s; }

@keyframes blink {
  0%, 80%, 100% { opacity: 0; }
  40% { opacity: 1; }
}

/* 响应式 */
@media (max-width: 768px) {
  .sidebar {
    width: 60px;
  }
  
  .sidebar-header h2,
  .session-title {
    display: none;
  }
  
  .session-item {
    justify-content: center;
    padding: 10px;
  }
  
  .btn-delete {
    display: none;
  }
  
  .message-content {
    max-width: 85%;
  }
}
</style>
