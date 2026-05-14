<template>
  <div class="app">
    <!-- 悬浮球 -->
    <div 
      v-if="!isOpen" 
      class="floating-ball" 
      @click="openChat"
      title="问答助手"
    >
      <span class="ball-icon">🤖</span>
      <span v-if="unreadCount > 0" class="unread-badge">{{ unreadCount }}</span>
    </div>

    <!-- 聊天窗口 -->
    <div v-if="isOpen" class="chat-window">
      <!-- 左侧工具栏 -->
      <div class="sidebar">
        <div class="sidebar-top">
          <button 
            class="sidebar-btn" 
            @click="createSession"
            title="新建会话"
          >
            <span class="btn-icon">✏️</span>
          </button>
          <button 
            class="sidebar-btn"
            :class="{ active: showHistory }"
            @click="toggleHistory"
            title="历史会话"
          >
            <span class="btn-icon">📚</span>
          </button>
        </div>
      </div>

      <!-- 历史会话浮层 -->
      <transition name="slide">
        <div v-if="showHistory" class="history-overlay" @click.self="showHistory = false">
          <div class="history-panel">
            <div class="panel-header">
              <span>历史会话</span>
              <button class="btn-close-panel" @click="showHistory = false">✕</button>
            </div>
            <div class="panel-content">
              <div
                v-for="session in sessions"
                :key="session.id"
                :class="['session-item', { active: currentSessionId === session.id }]"
                @click="selectSession(session.id)"
              >
                <div class="session-icon">💬</div>
                <div class="session-info">
                  <span class="session-title">{{ session.title || '新会话' }}</span>
                  <span class="session-time">{{ formatTime(session.created_at) }}</span>
                </div>
                <button class="btn-delete" @click.stop="deleteSession(session.id)">🗑</button>
              </div>
              <div v-if="sessions.length === 0" class="empty-hint">
                <div class="empty-icon">📝</div>
                <p>暂无历史会话</p>
                <p class="empty-sub">开始对话后将自动保存</p>
              </div>
            </div>
          </div>
        </div>
      </transition>

      <!-- 主聊天区 -->
      <div class="chat-main">
        <!-- 标题栏 -->
        <div class="chat-header">
          <div class="header-left">
            <span class="header-icon">🤖</span>
            <span class="header-title">{{ currentSessionTitle }}</span>
          </div>
          <button class="btn-close" @click="closeChat">✕</button>
        </div>

        <!-- 消息区域 -->
        <div class="messages-container">
          <div class="messages" ref="messagesContainer">
            <!-- 欢迎页 -->
            <div v-if="messages.length === 0" class="welcome">
              <div class="welcome-icon">🤖</div>
              <p class="welcome-title">你好！我是问答助手</p>
              <p class="welcome-sub">有什么可以帮你的？</p>
              <div class="welcome-hints">
                <div class="hint-item" @click="sendQuickMessage('查询订单 ORD-20260515-001')">
                  <span class="hint-icon">📦</span>
                  <span>查询订单</span>
                </div>
                <div class="hint-item" @click="sendQuickMessage('今天天气怎么样？')">
                  <span class="hint-icon">🌤</span>
                  <span>天气查询</span>
                </div>
                <div class="hint-item" @click="sendQuickMessage('帮我写一首诗')">
                  <span class="hint-icon">✍️</span>
                  <span>创意写作</span>
                </div>
              </div>
            </div>
            
            <!-- 消息列表 -->
            <div
              v-for="(msg, index) in messages"
              :key="index"
              :class="['message', msg.role]"
            >
              <div class="message-avatar">
                {{ msg.role === 'user' ? '👤' : '🤖' }}
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
            
            <!-- 加载状态 -->
            <div v-if="isLoading" class="message assistant loading">
              <div class="message-avatar">🤖</div>
              <div class="message-content">
                <span class="typing">思考中<span class="dot">.</span><span class="dot">.</span><span class="dot">.</span></span>
              </div>
            </div>
          </div>
        </div>

        <!-- 输入区域 -->
        <div class="input-area">
          <div class="input-wrapper">
            <textarea
              v-model="userInput"
              @keydown.enter.exact.prevent="sendMessage"
              placeholder="输入消息..."
              :disabled="isLoading"
              rows="1"
              ref="inputRef"
            ></textarea>
            <button 
              class="btn-send" 
              @click="sendMessage" 
              :disabled="isLoading || !userInput.trim()"
            >
              <span v-if="isLoading">⏳</span>
              <span v-else>➤</span>
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import { ref, computed, onMounted, nextTick } from 'vue'

export default {
  name: 'App',
  setup() {
    // UI 状态
    const isOpen = ref(false)
    const showHistory = ref(false)
    const unreadCount = ref(0)
    
    // 数据状态
    const sessions = ref([])
    const currentSessionId = ref('')
    const messages = ref([])
    const userInput = ref('')
    const isLoading = ref(false)
    const messagesContainer = ref(null)
    const inputRef = ref(null)
    
    let currentAbortController = null
    const API_BASE = '/api'

    // 计算属性：当前会话标题
    const currentSessionTitle = computed(() => {
      if (!currentSessionId.value) return '问答助手'
      const session = sessions.value.find(s => s.id === currentSessionId.value)
      return session?.title || '问答助手'
    })

    // 打开聊天
    const openChat = () => {
      isOpen.value = true
      showHistory.value = false
      unreadCount.value = 0
      loadSessions()
      nextTick(() => inputRef.value?.focus())
    }

    // 关闭聊天
    const closeChat = () => {
      isOpen.value = false
      showHistory.value = false
    }

    // 切换历史面板
    const toggleHistory = () => {
      showHistory.value = !showHistory.value
      if (showHistory.value) loadSessions()
    }

    // 加载会话列表
    const loadSessions = async () => {
      try {
        const res = await fetch(`${API_BASE}/sessions`)
        sessions.value = await res.json()
      } catch (e) {
        console.error('加载会话失败:', e)
      }
    }

    // 创建新会话（仅清空界面，不实际创建）
    const createSession = () => {
      currentSessionId.value = ''
      messages.value = []
      showHistory.value = false
      nextTick(() => inputRef.value?.focus())
    }

    // 选择会话
    const selectSession = async (sessionId) => {
      currentSessionId.value = sessionId
      showHistory.value = false
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

    // 确保有会话（发送消息时才创建）
    const ensureSession = async () => {
      if (currentSessionId.value) return
      
      try {
        const res = await fetch(`${API_BASE}/sessions`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' }
        })
        const session = await res.json()
        sessions.value.unshift(session)
        currentSessionId.value = session.id
      } catch (e) {
        console.error('创建会话失败:', e)
      }
    }

    // 发送快速消息
    const sendQuickMessage = async (text) => {
      userInput.value = text
      await sendMessage()
    }

    // 发送消息
    const sendMessage = async () => {
      const text = userInput.value.trim()
      if (!text || isLoading.value) return

      // 发送消息时才创建会话
      await ensureSession()

      messages.value.push({ role: 'user', content: text })
      userInput.value = ''
      isLoading.value = true
      await nextTick()
      scrollToBottom()

      let assistantContent = ''
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

        if (!res.ok) throw new Error(`HTTP ${res.status}`)

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
              try {
                const parsed = JSON.parse(data)
                
                if (eventType === 'session') {
                  currentSessionId.value = parsed.session_id
                  loadSessions()
                } else if (eventType === 'content') {
                  assistantContent += parsed.content
                  updateAssistantMessage(assistantContent)
                } else if (eventType === 'tool_call') {
                  assistantContent += '\n\n🔧 正在查询...'
                  updateAssistantMessage(assistantContent)
                } else if (eventType === 'tool_result') {
                  assistantContent = assistantContent.replace('\n\n🔧 正在查询...', '')
                  updateAssistantMessage(assistantContent, parsed.result)
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
        if (toolResult) lastMsg.tool_result = toolResult
      } else {
        messages.value.push({ role: 'assistant', content, tool_result: toolResult })
      }
      nextTick(() => scrollToBottom())
    }

    // 停止生成
    const stopGeneration = () => {
      if (currentAbortController) currentAbortController.abort()
    }

    // 滚动到底部
    const scrollToBottom = () => {
      if (messagesContainer.value) {
        messagesContainer.value.scrollTop = messagesContainer.value.scrollHeight
      }
    }

    // 格式化内容
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

    // 格式化时间
    const formatTime = (ts) => {
      if (!ts) return ''
      const d = new Date(ts)
      return `${d.getMonth() + 1}/${d.getDate()} ${d.getHours()}:${String(d.getMinutes()).padStart(2, '0')}`
    }

    onMounted(() => {})

    return {
      isOpen,
      showHistory,
      unreadCount,
      sessions,
      currentSessionId,
      currentSessionTitle,
      messages,
      userInput,
      isLoading,
      messagesContainer,
      inputRef,
      openChat,
      closeChat,
      toggleHistory,
      createSession,
      selectSession,
      deleteSession,
      sendQuickMessage,
      sendMessage,
      stopGeneration,
      formatContent,
      formatTime
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
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  min-height: 100vh;
}

#app {
  min-height: 100vh;
}

.app {
  min-height: 100vh;
}

/* ===== 悬浮球 ===== */
.floating-ball {
  position: fixed;
  bottom: 24px;
  right: 24px;
  width: 56px;
  height: 56px;
  background: linear-gradient(135deg, #4299e1 0%, #667eea 100%);
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  box-shadow: 0 4px 20px rgba(66, 153, 225, 0.4);
  transition: all 0.3s ease;
  z-index: 1000;
}

.floating-ball:hover {
  transform: scale(1.1);
  box-shadow: 0 6px 30px rgba(66, 153, 225, 0.5);
}

.ball-icon {
  font-size: 26px;
}

.unread-badge {
  position: absolute;
  top: -4px;
  right: -4px;
  background: #e53e3e;
  color: white;
  font-size: 11px;
  font-weight: bold;
  min-width: 18px;
  height: 18px;
  border-radius: 9px;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 0 5px;
}

/* ===== 聊天窗口 ===== */
.chat-window {
  position: fixed;
  bottom: 24px;
  right: 24px;
  width: 420px;
  height: 600px;
  background: white;
  border-radius: 16px;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.15);
  display: flex;
  overflow: hidden;
  z-index: 1000;
  animation: slideUp 0.3s ease;
  border: 1px solid rgba(255, 255, 255, 0.2);
}

@keyframes slideUp {
  from { opacity: 0; transform: translateY(20px); }
  to { opacity: 1; transform: translateY(0); }
}

/* ===== 左侧工具栏 ===== */
.sidebar {
  width: 52px;
  background: linear-gradient(180deg, #4299e1 0%, #3182ce 100%);
  display: flex;
  flex-direction: column;
  padding: 14px 0;
  flex-shrink: 0;
}

.sidebar-top {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
}

.sidebar-btn {
  width: 40px;
  height: 40px;
  background: rgba(255, 255, 255, 0.15);
  border: none;
  border-radius: 10px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s;
}

.sidebar-btn:hover {
  background: rgba(255, 255, 255, 0.25);
}

.sidebar-btn.active {
  background: rgba(255, 255, 255, 0.3);
}

.btn-icon {
  font-size: 18px;
}

/* ===== 历史会话浮层 ===== */
.history-overlay {
  position: absolute;
  top: 0;
  left: 52px;
  width: 280px;
  height: 100%;
  background: rgba(0, 0, 0, 0.3);
  z-index: 10;
}

.history-panel {
  width: 100%;
  height: 100%;
  background: #f7fafc;
  display: flex;
  flex-direction: column;
}

.panel-header {
  padding: 16px;
  font-weight: 600;
  font-size: 15px;
  color: #2d3748;
  background: white;
  border-bottom: 1px solid #e2e8f0;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.btn-close-panel {
  width: 28px;
  height: 28px;
  background: #edf2f7;
  border: none;
  border-radius: 6px;
  cursor: pointer;
  font-size: 14px;
  color: #718096;
}

.btn-close-panel:hover {
  background: #e2e8f0;
}

.panel-content {
  flex: 1;
  overflow-y: auto;
  padding: 12px;
}

.session-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px;
  background: white;
  border: 1px solid #e2e8f0;
  border-radius: 10px;
  cursor: pointer;
  margin-bottom: 8px;
  transition: all 0.2s;
}

.session-item:hover {
  border-color: #4299e1;
  background: #ebf8ff;
}

.session-item.active {
  border-color: #4299e1;
  background: #ebf8ff;
}

.session-icon {
  font-size: 20px;
  flex-shrink: 0;
}

.session-info {
  flex: 1;
  min-width: 0;
}

.session-title {
  display: block;
  font-size: 13px;
  color: #2d3748;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.session-time {
  display: block;
  font-size: 11px;
  color: #a0aec0;
  margin-top: 2px;
}

.btn-delete {
  background: none;
  border: none;
  cursor: pointer;
  font-size: 14px;
  padding: 4px;
  opacity: 0;
  transition: opacity 0.2s;
}

.session-item:hover .btn-delete {
  opacity: 1;
}

.btn-delete:hover {
  opacity: 1 !important;
}

.empty-hint {
  text-align: center;
  color: #a0aec0;
  padding: 40px 20px;
}

.empty-icon {
  font-size: 40px;
  margin-bottom: 12px;
}

.empty-hint p {
  font-size: 14px;
  margin-bottom: 4px;
}

.empty-sub {
  font-size: 12px !important;
  color: #cbd5e0 !important;
}

/* ===== 动画 ===== */
.slide-enter-active,
.slide-leave-active {
  transition: all 0.25s ease;
}

.slide-enter-from,
.slide-leave-to {
  opacity: 0;
  transform: translateX(-20px);
}

/* ===== 主聊天区 ===== */
.chat-main {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
}

/* ===== 标题栏 ===== */
.chat-header {
  background: linear-gradient(135deg, #4299e1 0%, #667eea 100%);
  color: white;
  padding: 14px 16px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-shrink: 0;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 10px;
}

.header-icon {
  font-size: 22px;
}

.header-title {
  font-size: 15px;
  font-weight: 600;
}

.btn-close {
  width: 28px;
  height: 28px;
  background: rgba(255, 255, 255, 0.2);
  border: none;
  border-radius: 6px;
  color: white;
  font-size: 14px;
  cursor: pointer;
}

.btn-close:hover {
  background: rgba(255, 255, 255, 0.3);
}

/* ===== 消息区域 ===== */
.messages-container {
  flex: 1;
  min-height: 0;
  overflow: hidden;
}

.messages {
  height: 100%;
  overflow-y: auto;
  padding: 16px;
}

.welcome {
  text-align: center;
  color: #718096;
  padding: 40px 20px;
}

.welcome-icon {
  font-size: 56px;
  margin-bottom: 16px;
}

.welcome-title {
  font-size: 18px;
  font-weight: 600;
  color: #2d3748;
  margin-bottom: 8px;
}

.welcome-sub {
  font-size: 14px;
  margin-bottom: 24px;
}

.welcome-hints {
  display: flex;
  flex-direction: column;
  gap: 10px;
  max-width: 260px;
  margin: 0 auto;
}

.hint-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 16px;
  background: #f7fafc;
  border: 1px solid #e2e8f0;
  border-radius: 12px;
  cursor: pointer;
  font-size: 14px;
  transition: all 0.2s;
}

.hint-item:hover {
  background: #ebf8ff;
  border-color: #4299e1;
}

.hint-icon {
  font-size: 18px;
}

.message {
  display: flex;
  margin-bottom: 16px;
  animation: fadeIn 0.3s;
}

@keyframes fadeIn {
  from { opacity: 0; transform: translateY(8px); }
  to { opacity: 1; transform: translateY(0); }
}

.message.user {
  flex-direction: row-reverse;
}

.message-avatar {
  width: 36px;
  height: 36px;
  border-radius: 50%;
  background: #e2e8f0;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 18px;
  flex-shrink: 0;
}

.message.user .message-avatar {
  background: #4299e1;
}

.message-content {
  max-width: 75%;
  margin: 0 10px;
}

.message-text {
  padding: 10px 14px;
  border-radius: 14px;
  background: #e2e8f0;
  line-height: 1.5;
  font-size: 14px;
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
  font-size: 13px;
}

.message.assistant .message-text code {
  background: #edf2f7;
}

.tool-call {
  margin-top: 8px;
  padding: 8px 12px;
  background: #fef3c7;
  border-radius: 8px;
  font-size: 12px;
  color: #92400e;
}

.tool-result {
  margin-top: 8px;
  background: #f0fff4;
  border: 1px solid #9ae6b4;
  border-radius: 8px;
  overflow: hidden;
}

.tool-result-header {
  padding: 6px 10px;
  background: #9ae6b4;
  font-size: 11px;
  font-weight: 600;
  color: #22543d;
}

.tool-result pre {
  padding: 10px;
  font-size: 12px;
  white-space: pre-wrap;
  word-break: break-word;
  color: #2d3748;
  margin: 0;
}

/* ===== 输入区域 ===== */
.input-area {
  padding: 12px 16px;
  border-top: 1px solid #e2e8f0;
  background: white;
}

.input-wrapper {
  display: flex;
  gap: 8px;
  align-items: flex-end;
}

.input-wrapper textarea {
  flex: 1;
  padding: 10px 14px;
  border: 1px solid #e2e8f0;
  border-radius: 20px;
  resize: none;
  font-size: 14px;
  outline: none;
  font-family: inherit;
  max-height: 100px;
}

.input-wrapper textarea:focus {
  border-color: #4299e1;
}

.btn-send {
  width: 40px;
  height: 40px;
  background: #4299e1;
  color: white;
  border: none;
  border-radius: 50%;
  cursor: pointer;
  font-size: 16px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.btn-send:hover:not(:disabled) {
  background: #3182ce;
}

.btn-send:disabled {
  background: #a0aec0;
  cursor: not-allowed;
}

/* ===== 加载动画 ===== */
.typing .dot {
  animation: blink 1.4s infinite both;
}

.dot:nth-child(2) { animation-delay: 0.2s; }
.dot:nth-child(3) { animation-delay: 0.4s; }

@keyframes blink {
  0%, 80%, 100% { opacity: 0; }
  40% { opacity: 1; }
}

/* ===== 响应式 ===== */
@media (max-width: 480px) {
  .chat-window {
    bottom: 0;
    right: 0;
    left: 0;
    width: 100%;
    height: 100%;
    border-radius: 0;
  }
  
  .history-overlay {
    width: 260px;
  }
  
  .floating-ball {
    bottom: 16px;
    right: 16px;
  }
}
</style>
