<template>
  <div class="app">
    <!-- 聊天窗口 - 直接居中显示 -->
    <div class="chat-window centered" :class="{ 'fullscreen-mode': isFullscreen }">
      <div class="window-glow"></div>

      <!-- 历史会话抽屉 -->
      <transition name="drawer">
        <div v-if="showHistory" class="history-drawer">
          <div class="drawer-header">
            <h3>对话历史</h3>
            <button class="btn-icon" @click="showHistory = false">
              <svg class="icon-sm" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M6 18L18 6M6 6l12 12"/>
              </svg>
            </button>
          </div>
          <div class="drawer-content">
            <div
              v-for="session in sessions"
              :key="session.id"
              :class="['session-item', { active: currentSessionId === session.id }]"
              @click="selectSession(session.id)"
            >
              <div class="session-icon">
                <svg class="icon-sm" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z"/>
                </svg>
              </div>
              <div class="session-info">
                <span class="session-title">{{ session.title || '新会话' }}</span>
                <span class="session-time">{{ formatTime(session.created_at) }}</span>
              </div>
              <button class="btn-delete" @click.stop="deleteSession(session.id)">
                <svg class="icon-xs" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"/>
                </svg>
              </button>
            </div>
            <div v-if="sessions.length === 0" class="empty-hint">
              <div class="empty-icon-wrap">
                <svg class="icon-lg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
                  <path d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z"/>
                </svg>
              </div>
              <p>暂无对话记录</p>
              <p class="empty-sub">开始对话后将自动保存</p>
            </div>
          </div>
        </div>
      </transition>

      <!-- 主聊天区 -->
      <div class="chat-main">
        <!-- 标题栏 -->
        <div class="chat-header">
          <div class="header-left">
            <div class="header-avatar">
              <span>🤖</span>
            </div>
            <div class="header-info">
              <span class="header-title">{{ currentSessionTitle }}</span>
              <span class="header-status">
                <span class="status-dot" :class="{ online: !isLoading }"></span>
                {{ isLoading ? '正在思考...' : '在线' }}
              </span>
            </div>
          </div>
          <div class="header-actions">
            <button class="btn-header" @click="createSession" title="新建会话">
              <svg class="icon-sm" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M12 5v14M5 12h14"/>
              </svg>
            </button>
            <button class="btn-header" @click="toggleHistory" :class="{ active: showHistory }" title="历史会话">
              <svg class="icon-sm" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"/>
              </svg>
            </button>
            <button class="btn-header" @click="toggleFullscreen" :title="isFullscreen ? '退出全屏' : '全屏'">
              <svg v-if="!isFullscreen" class="icon-sm" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M8 3H5a2 2 0 00-2 2v3m18 0V5a2 2 0 00-2-2h-3m0 18h3a2 2 0 002-2v-3M3 16v3a2 2 0 002 2h3"/>
              </svg>
              <svg v-else class="icon-sm" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M8 3v3a2 2 0 01-2 2H3m18 0h-3a2 2 0 01-2-2V3m0 18v-3a2 2 0 012-2h3M3 16h3a2 2 0 012 2v3"/>
              </svg>
            </button>
            <button class="btn-header btn-close" @click="closeChat" title="关闭">
              <svg class="icon-sm" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M6 18L18 6M6 6l12 12"/>
              </svg>
            </button>
          </div>
        </div>

        <!-- 消息区域 -->
        <div class="messages-container">
          <div class="messages" ref="messagesContainer">
            <!-- 欢迎页 -->
            <div v-if="messages.length === 0" class="welcome">
              <div class="welcome-glow"></div>
              <div class="welcome-icon">🤖</div>
              <h2 class="welcome-title">你好！我是问答助手</h2>
              <p class="welcome-sub">有什么可以帮你的？</p>
              <div class="welcome-hints">
                <button class="hint-item" @click="sendQuickMessage('查询订单 ORD-20260515-001')">
                  <span class="hint-icon">📦</span>
                  <span>查询订单</span>
                </button>
                <button class="hint-item" @click="sendQuickMessage('今天天气怎么样？')">
                  <span class="hint-icon">🌤</span>
                  <span>天气查询</span>
                </button>
                <button class="hint-item" @click="sendQuickMessage('帮我写一首诗')">
                  <span class="hint-icon">✍️</span>
                  <span>创意写作</span>
                </button>
              </div>
            </div>

            <!-- 消息列表 -->
            <div
              v-for="(msg, index) in messages"
              :key="index"
              :class="['message', msg.role]"
              :style="{ animationDelay: `${index * 0.05}s` }"
            >
              <div class="message-avatar">
                {{ msg.role === 'user' ? '👤' : '🤖' }}
              </div>
              <div class="message-content">
                <div class="message-text" v-html="formatContent(msg.content)"></div>
                <div v-if="msg.tool_call" class="tool-call">
                  <span class="tool-icon">⚡</span>
                  调用工具: {{ msg.tool_call.name }}
                </div>
                <div v-if="msg.tool_result" class="tool-result">
                  <div class="tool-result-header">
                    <svg class="icon-xs" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                      <path d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-6 9l2 2 4-4"/>
                    </svg>
                    查询结果
                  </div>
                  <pre>{{ msg.tool_result }}</pre>
                </div>
              </div>
            </div>

            <!-- 加载状态 -->
            <div v-if="isLoading" class="message assistant loading">
              <div class="message-avatar">🤖</div>
              <div class="message-content">
                <div class="typing-indicator">
                  <span class="typing-dot"></span>
                  <span class="typing-dot"></span>
                  <span class="typing-dot"></span>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- 输入区域 -->
        <div class="input-area">
          <div class="input-wrapper">
            <div class="textarea-wrap">
              <div class="resize-handle" @mousedown="startResize"></div>
              <textarea
                v-model="userInput"
                @input="autoResize"
                @keydown.enter.exact.prevent="sendMessage"
                placeholder="输入消息..."
                :disabled="isLoading"
                rows="1"
                ref="inputRef"
              ></textarea>
            </div>
            <button
              v-if="isLoading"
              class="btn-send btn-stop"
              @click="stopGeneration"
              title="停止生成"
            >
              <svg class="icon-sm" viewBox="0 0 24 24" fill="currentColor">
                <rect x="6" y="6" width="12" height="12" rx="2"/>
              </svg>
            </button>
            <button
              v-else
              class="btn-send"
              @click="sendMessage"
              :disabled="!userInput.trim()"
              title="发送"
            >
              <svg class="icon-sm" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M22 2L11 13M22 2l-7 20-4-9-9-4 20-7z"/>
              </svg>
            </button>
          </div>
          <div class="input-hint">拖动上边缘调整高度 · Enter 发送</div>
        </div>
      </div>
    </div>

    <!-- 全屏模式背景 -->
    <div v-if="isFullscreen" class="fullscreen-backdrop"></div>
    
    <!-- 历史抽屉遮罩层 -->
    <transition name="fade">
      <div v-if="showHistory" class="drawer-overlay" @click="showHistory = false"></div>
    </transition>
  </div>
</template>

<script>
import { ref, computed, nextTick } from 'vue'
import MarkdownIt from 'markdown-it'

// 配置 markdown-it
const md = new MarkdownIt({
  html: true,
  linkify: true,
  typographer: true,
  breaks: true
})

export default {
  name: 'App',
  setup() {
    // UI 状态
    const isOpen = ref(false)
    const showHistory = ref(false)
    const unreadCount = ref(0)
    const isFullscreen = ref(false)

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

    // 切换网页内全屏
    const toggleFullscreen = () => {
      isFullscreen.value = !isFullscreen.value
    }

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
      isFullscreen.value = false
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

    // 发送快速消息
    const sendQuickMessage = async (text) => {
      userInput.value = text
      await sendMessage()
    }

    // 发送消息（后端自动创建会话）
    const sendMessage = async () => {
      const text = userInput.value.trim()
      if (!text || isLoading.value) return

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
                  assistantContent += '\n\n⚡ 正在查询...'
                  updateAssistantMessage(assistantContent)
                } else if (eventType === 'tool_result') {
                  assistantContent = assistantContent.replace('\n\n⚡ 正在查询...', '')
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

    // 格式化内容 - 使用 markdown-it 库渲染 Markdown
    const formatContent = (content) => {
      if (!content) return ''
      return md.render(content)
    }

    // 格式化时间
    const formatTime = (ts) => {
      if (!ts) return ''
      const d = new Date(ts)
      return `${d.getMonth() + 1}/${d.getDate()} ${d.getHours()}:${String(d.getMinutes()).padStart(2, '0')}`
    }

    // 自动调整输入框高度
    const autoResize = () => {
      const el = inputRef.value
      if (!el) return
      el.style.height = 'auto'
      el.style.height = Math.min(el.scrollHeight, 340) + 'px'
    }

    // 拖动调整输入框高度
    let isResizing = false
    let startY = 0
    let startHeight = 0

    const startResize = (e) => {
      isResizing = true
      startY = e.clientY
      startHeight = inputRef.value?.offsetHeight || 52
      document.addEventListener('mousemove', doResize)
      document.addEventListener('mouseup', stopResize)
      document.body.style.cursor = 'ns-resize'
      document.body.style.userSelect = 'none'
    }

    const doResize = (e) => {
      if (!isResizing || !inputRef.value) return
      const delta = startY - e.clientY
      const newHeight = Math.max(52, Math.min(240, startHeight + delta))
      inputRef.value.style.height = newHeight + 'px'
    }

    const stopResize = () => {
      isResizing = false
      document.removeEventListener('mousemove', doResize)
      document.removeEventListener('mouseup', stopResize)
      document.body.style.cursor = ''
      document.body.style.userSelect = ''
    }

    return {
      isOpen,
      showHistory,
      unreadCount,
      isFullscreen,
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
      toggleFullscreen,
      createSession,
      selectSession,
      deleteSession,
      sendQuickMessage,
      sendMessage,
      stopGeneration,
      autoResize,
      startResize,
      formatContent,
      formatTime
    }
  }
}
</script>

<style>
/* Fonts loaded via index.html with preconnect for better performance */

:root {
  --bg-primary: #0a0a12;
  --bg-secondary: #12121e;
  --bg-elevated: #1a1a2e;
  --bg-glass: rgba(18, 18, 30, 0.85);
  --border-glass: rgba(255, 255, 255, 0.08);
  --text-primary: #f0f0f5;
  --text-secondary: #8888a0;
  --text-muted: #55556a;
  --accent-primary: #7c5cfc;
  --accent-secondary: #5ce1e6;
  --accent-gradient: linear-gradient(135deg, #7c5cfc 0%, #5ce1e6 100%);
  --danger: #ff4757;
  --success: #2ed573;
  --warning: #ffa502;
  --shadow-glow: 0 0 60px rgba(124, 92, 252, 0.15);
  --radius-sm: 8px;
  --radius-md: 12px;
  --radius-lg: 20px;
  --radius-xl: 28px;
}

* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

body {
  font-family: 'Plus Jakarta Sans', -apple-system, sans-serif;
  background: var(--bg-primary);
  min-height: 100vh;
  color: var(--text-primary);
}

body::before {
  content: '';
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background:
    radial-gradient(ellipse 80% 50% at 20% 40%, rgba(124, 92, 252, 0.08) 0%, transparent 60%),
    radial-gradient(ellipse 60% 40% at 80% 60%, rgba(92, 225, 230, 0.06) 0%, transparent 50%);
  pointer-events: none;
  z-index: 0;
}

#app {
  min-height: 100vh;
  position: relative;
  z-index: 1;
}

.app {
  min-height: 100vh;
}

/* ===== Icons ===== */
.icon { width: 20px; height: 20px; }
.icon-sm { width: 16px; height: 16px; }
.icon-xs { width: 14px; height: 14px; }
.icon-lg { width: 48px; height: 48px; }

/* ===== 聊天窗口 ===== */
.floating-ball {
  position: fixed;
  bottom: 28px;
  right: 28px;
  width: 60px;
  height: 60px;
  background: var(--accent-gradient);
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  z-index: 1000;
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
}

.ball-glow {
  position: absolute;
  inset: -8px;
  background: var(--accent-gradient);
  border-radius: 50%;
  filter: blur(20px);
  opacity: 0.4;
  transition: opacity 0.3s;
}

.floating-ball:hover {
  transform: scale(1.08);
}

.floating-ball:hover .ball-glow {
  opacity: 0.6;
}

.ball-icon {
  font-size: 28px;
  position: relative;
  z-index: 1;
  filter: drop-shadow(0 2px 4px rgba(0,0,0,0.2));
}

.unread-badge {
  position: absolute;
  top: -2px;
  right: -2px;
  background: var(--danger);
  color: white;
  font-size: 11px;
  font-weight: 700;
  min-width: 20px;
  height: 20px;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 0 6px;
  border: 2px solid var(--bg-primary);
  z-index: 2;
  animation: badgePop 0.3s cubic-bezier(0.68, -0.55, 0.265, 1.55);
}

@keyframes badgePop {
  0% { transform: scale(0); }
  100% { transform: scale(1); }
}

/* ===== 聊天窗口 ===== */
.chat-window {
  width: 720px;
  height: 800px;
  background: var(--bg-glass);
  backdrop-filter: blur(24px);
  -webkit-backdrop-filter: blur(24px);
  border-radius: var(--radius-xl);
  box-shadow:
    var(--shadow-glow),
    0 24px 80px rgba(0, 0, 0, 0.5),
    inset 0 1px 0 var(--border-glass);
  display: flex;
  overflow: hidden;
  z-index: 1000;
  animation: windowAppear 0.4s cubic-bezier(0.4, 0, 0.2, 1);
  border: 1px solid var(--border-glass);
}

/* 居中模式 */
.chat-window.centered {
  position: fixed;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
}

.window-glow {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 120px;
  background: linear-gradient(180deg, rgba(124, 92, 252, 0.08) 0%, transparent 100%);
  pointer-events: none;
}

@keyframes windowAppear {
  from {
    opacity: 0;
    transform: translate(-50%, -45%) scale(0.96);
  }
  to {
    opacity: 1;
    transform: translate(-50%, -50%) scale(1);
  }
}

/* ===== 历史记录抽屉 ===== */
.history-drawer {
  position: absolute;
  top: 0;
  right: 0;
  width: 320px;
  height: 100%;
  background: var(--bg-secondary);
  border-left: 1px solid var(--border-glass);
  z-index: 20;
  display: flex;
  flex-direction: column;
  box-shadow: -8px 0 32px rgba(0, 0, 0, 0.3);
}

.drawer-header {
  padding: 18px 20px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  border-bottom: 1px solid var(--border-glass);
  background: rgba(0, 0, 0, 0.2);
}

.drawer-header h3 {
  font-family: 'Outfit', sans-serif;
  font-weight: 600;
  font-size: 15px;
  color: var(--text-primary);
  margin: 0;
}

.btn-icon {
  width: 32px;
  height: 32px;
  background: transparent;
  border: 1px solid var(--border-glass);
  border-radius: var(--radius-sm);
  cursor: pointer;
  color: var(--text-secondary);
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s;
}

.btn-icon:hover {
  background: rgba(255, 255, 255, 0.05);
  color: var(--text-primary);
}

.drawer-content {
  flex: 1;
  overflow-y: auto;
  padding: 12px;
}

.session-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 14px;
  background: transparent;
  border: 1px solid transparent;
  border-radius: var(--radius-md);
  cursor: pointer;
  margin-bottom: 6px;
  transition: all 0.2s;
}

.session-item:hover {
  background: rgba(255, 255, 255, 0.03);
  border-color: var(--border-glass);
}

.session-item.active {
  background: rgba(124, 92, 252, 0.1);
  border-color: rgba(124, 92, 252, 0.2);
}

.session-icon {
  width: 36px;
  height: 36px;
  background: rgba(255, 255, 255, 0.05);
  border-radius: var(--radius-sm);
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--text-secondary);
  flex-shrink: 0;
}

.session-info {
  flex: 1;
  min-width: 0;
}

.session-title {
  display: block;
  font-size: 13px;
  font-weight: 500;
  color: var(--text-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.session-time {
  display: block;
  font-size: 11px;
  color: var(--text-muted);
  margin-top: 3px;
}

.btn-delete {
  background: none;
  border: none;
  cursor: pointer;
  color: var(--text-muted);
  padding: 8px;
  border-radius: var(--radius-sm);
  opacity: 0;
  transition: all 0.2s;
}

.btn-delete:hover {
  background: rgba(255, 71, 87, 0.1);
  color: var(--danger);
}

.session-item:hover .btn-delete {
  opacity: 1;
}

.empty-hint {
  text-align: center;
  color: var(--text-muted);
  padding: 48px 24px;
}

.empty-icon-wrap {
  width: 72px;
  height: 72px;
  margin: 0 auto 20px;
  background: rgba(255, 255, 255, 0.03);
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--text-muted);
}

.empty-hint p {
  font-size: 14px;
  margin-bottom: 6px;
  color: var(--text-secondary);
}

.empty-sub {
  font-size: 12px !important;
  color: var(--text-muted) !important;
}

/* ===== 抽屉动画 ===== */
.drawer-enter-active,
.drawer-leave-active {
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
}

.drawer-enter-from,
.drawer-leave-to {
  opacity: 0;
  transform: translateX(100%);
}

/* ===== 遮罩层 ===== */
.drawer-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.5);
  z-index: 15;
  backdrop-filter: blur(2px);
}

.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.3s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}

/* ===== 主聊天区 ===== */
.chat-main {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
  position: relative;
}

/* ===== 标题栏 ===== */
.chat-header {
  background: rgba(0, 0, 0, 0.3);
  border-bottom: 1px solid var(--border-glass);
  padding: 14px 18px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-shrink: 0;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 12px;
}

.header-avatar {
  width: 40px;
  height: 40px;
  background: var(--accent-gradient);
  border-radius: var(--radius-md);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 20px;
}

.header-info {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.header-title {
  font-family: 'Outfit', sans-serif;
  font-size: 15px;
  font-weight: 600;
  color: var(--text-primary);
}

.header-status {
  font-size: 11px;
  color: var(--text-muted);
  display: flex;
  align-items: center;
  gap: 5px;
}

.status-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--text-muted);
  transition: background 0.3s;
}

.status-dot.online {
  background: var(--success);
  box-shadow: 0 0 8px rgba(46, 213, 115, 0.5);
}

.btn-close {
  width: 32px;
  height: 32px;
  background: transparent;
  border: 1px solid var(--border-glass);
  border-radius: var(--radius-sm);
  color: var(--text-secondary);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s;
}

.btn-close:hover {
  background: rgba(255, 71, 87, 0.1);
  border-color: rgba(255, 71, 87, 0.3);
  color: var(--danger);
}

.header-actions {
  display: flex;
  align-items: center;
  gap: 6px;
}

.btn-header {
  width: 32px;
  height: 32px;
  background: transparent;
  border: 1px solid var(--border-glass);
  border-radius: var(--radius-sm);
  color: var(--text-secondary);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s;
}

.btn-header:hover {
  background: rgba(255, 255, 255, 0.05);
  border-color: rgba(255, 255, 255, 0.15);
  color: var(--text-primary);
}

.btn-header.active {
  background: rgba(124, 92, 252, 0.15);
  border-color: rgba(124, 92, 252, 0.3);
  color: var(--accent-primary);
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
  padding: 20px;
  scroll-behavior: smooth;
}

/* Custom scrollbar */
.messages::-webkit-scrollbar {
  width: 6px;
}

.messages::-webkit-scrollbar-track {
  background: transparent;
}

.messages::-webkit-scrollbar-thumb {
  background: rgba(255, 255, 255, 0.1);
  border-radius: 3px;
}

.messages::-webkit-scrollbar-thumb:hover {
  background: rgba(255, 255, 255, 0.2);
}

.welcome {
  text-align: center;
  padding: 40px 24px;
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  flex: 1;
}

.welcome-glow {
  position: absolute;
  top: 30%;
  left: 50%;
  transform: translate(-50%, -50%);
  width: 200px;
  height: 200px;
  background: radial-gradient(circle, rgba(124, 92, 252, 0.15) 0%, transparent 70%);
  pointer-events: none;
}

.welcome-icon {
  font-size: 56px;
  margin-bottom: 16px;
  position: relative;
  animation: float 3s ease-in-out infinite;
}

@keyframes float {
  0%, 100% { transform: translateY(0); }
  50% { transform: translateY(-8px); }
}

.welcome-title {
  font-family: 'Outfit', sans-serif;
  font-size: 20px;
  font-weight: 600;
  color: var(--text-primary);
  margin-bottom: 6px;
  position: relative;
}

.welcome-sub {
  font-size: 14px;
  color: var(--text-secondary);
  margin-bottom: 24px;
  position: relative;
}

.welcome-hints {
  display: flex;
  flex-direction: column;
  gap: 10px;
  max-width: 280px;
  margin: 0 auto;
  position: relative;
}

.hint-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 14px 18px;
  background: rgba(255, 255, 255, 0.03);
  border: 1px solid var(--border-glass);
  border-radius: var(--radius-md);
  cursor: pointer;
  font-size: 14px;
  color: var(--text-primary);
  transition: all 0.2s;
}

.hint-item:hover {
  background: rgba(124, 92, 252, 0.1);
  border-color: rgba(124, 92, 252, 0.3);
  transform: translateX(4px);
}

.hint-icon {
  font-size: 18px;
}

.message {
  display: flex;
  margin-bottom: 20px;
  animation: messageSlide 0.4s cubic-bezier(0.4, 0, 0.2, 1);
  animation-fill-mode: both;
}

@keyframes messageSlide {
  from {
    opacity: 0;
    transform: translateY(12px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.message.user {
  flex-direction: row-reverse;
}

.message-avatar {
  width: 36px;
  height: 36px;
  border-radius: var(--radius-md);
  background: var(--bg-elevated);
  border: 1px solid var(--border-glass);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 16px;
  flex-shrink: 0;
}

.message.user .message-avatar {
  background: var(--accent-gradient);
  border: none;
}

.message-content {
  max-width: 80%;
  margin: 0 12px;
}

.chat-window.fullscreen-mode .message-content {
  max-width: 65%;
}

.message-text {
  padding: 14px 18px;
  border-radius: var(--radius-lg);
  background: var(--bg-elevated);
  border: 1px solid var(--border-glass);
  line-height: 1.6;
  font-size: 14px;
  word-break: break-word;
  color: var(--text-primary);
}

.message.user .message-text {
  background: var(--accent-primary);
  border: none;
  color: white;
}

/* Markdown 样式 */
.message-text h1,
.message-text h2,
.message-text h3,
.message-text h4,
.message-text h5,
.message-text h6 {
  color: var(--text-primary);
  margin: 1em 0 0.5em;
}

.message-text h1 { font-size: 1.5em; }
.message-text h2 { font-size: 1.3em; }
.message-text h3 { font-size: 1.1em; }

.message-text p {
  margin: 0.5em 0;
}

.message-text code {
  background: rgba(124, 92, 252, 0.2);
  padding: 2px 8px;
  border-radius: var(--radius-sm);
  font-family: 'JetBrains Mono', monospace;
  font-size: 13px;
  color: var(--accent-secondary);
}

.message.assistant .message-text code {
  background: rgba(92, 225, 230, 0.1);
  color: var(--accent-secondary);
}

.message-text pre {
  background: rgba(0, 0, 0, 0.3);
  border: 1px solid var(--border-glass);
  border-radius: var(--radius-md);
  padding: 12px;
  margin: 12px 0;
  overflow-x: auto;
}

.message-text pre code {
  background: none;
  padding: 0;
  font-size: 13px;
  color: var(--text-secondary);
}

.message-text table {
  width: 100%;
  border-collapse: collapse;
  margin: 12px 0;
  font-size: 13px;
}

.message-text th,
.message-text td {
  padding: 10px 14px;
  border: 1px solid var(--border-glass);
  text-align: left;
}

.message-text th {
  background: rgba(124, 92, 252, 0.1);
  font-weight: 600;
  color: var(--text-primary);
}

.message-text td {
  color: var(--text-secondary);
}

.message-text ul,
.message-text ol {
  margin: 0.5em 0;
  padding-left: 1.5em;
}

.message-text li {
  margin: 0.25em 0;
}

.message-text blockquote {
  border-left: 3px solid var(--accent-primary);
  padding-left: 12px;
  margin: 12px 0;
  color: var(--text-secondary);
}

.message-text hr {
  border: none;
  border-top: 1px solid var(--border-glass);
  margin: 16px 0;
}

.message-text a {
  color: var(--accent-secondary);
  text-decoration: none;
}

.message-text a:hover {
  text-decoration: underline;
}

.tool-call {
  margin-top: 10px;
  padding: 10px 14px;
  background: rgba(255, 165, 2, 0.1);
  border: 1px solid rgba(255, 165, 2, 0.2);
  border-radius: var(--radius-md);
  font-size: 13px;
  color: var(--warning);
  display: flex;
  align-items: center;
  gap: 8px;
}

.tool-icon {
  font-size: 14px;
}

.tool-result {
  margin-top: 10px;
  background: rgba(46, 213, 115, 0.05);
  border: 1px solid rgba(46, 213, 115, 0.2);
  border-radius: var(--radius-md);
  overflow: hidden;
}

.tool-result-header {
  padding: 10px 14px;
  background: rgba(46, 213, 115, 0.1);
  font-size: 12px;
  font-weight: 600;
  color: var(--success);
  display: flex;
  align-items: center;
  gap: 8px;
}

.tool-result pre {
  padding: 14px;
  font-size: 12px;
  font-family: 'JetBrains Mono', monospace;
  white-space: pre-wrap;
  word-break: break-word;
  color: var(--text-secondary);
  margin: 0;
  line-height: 1.5;
}

/* ===== 输入区域 ===== */
.input-area {
  padding: 14px 18px 10px;
  background: rgba(0, 0, 0, 0.2);
  border-top: 1px solid var(--border-glass);
}

.input-wrapper {
  display: flex;
  gap: 10px;
  align-items: flex-end;
}

.textarea-wrap {
  flex: 1;
  position: relative;
}

.resize-handle {
  position: absolute;
  top: 0;
  left: 16px;
  right: 16px;
  height: 8px;
  cursor: ns-resize;
  z-index: 1;
}

.resize-handle::before {
  content: '';
  position: absolute;
  top: 0;
  left: 50%;
  transform: translateX(-50%);
  width: 40px;
  height: 4px;
  background: rgba(255, 255, 255, 0.1);
  border-radius: 2px;
  transition: all 0.2s;
}

.resize-handle:hover::before {
  background: var(--accent-primary);
  width: 60px;
}

.textarea-wrap textarea {
  flex: 1;
  padding: 12px 18px;
  background: var(--bg-elevated);
  border: 1px solid var(--border-glass);
  border-radius: var(--radius-lg);
  resize: none;
  font-size: 14px;
  font-family: inherit;
  outline: none;
  color: var(--text-primary);
  min-height: 48px;
  max-height: 240px;
  line-height: 1.5;
  overflow-y: auto;
  transition: all 0.2s;
  width: 100%;
  display: flex;
  align-items: center;
}

.input-wrapper textarea::placeholder {
  color: var(--text-muted);
}

.input-wrapper textarea:focus {
  border-color: rgba(124, 92, 252, 0.4);
  box-shadow: 0 0 0 3px rgba(124, 92, 252, 0.1);
}

.btn-send {
  width: 48px;
  height: 48px;
  background: var(--accent-gradient);
  color: white;
  border: none;
  border-radius: 14px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  transition: all 0.2s;
  box-shadow: 0 4px 16px rgba(124, 92, 252, 0.3);
}

.btn-send:hover:not(:disabled) {
  transform: scale(1.05);
  box-shadow: 0 6px 24px rgba(124, 92, 252, 0.4);
}

.btn-send:disabled {
  background: var(--bg-elevated);
  box-shadow: none;
  cursor: not-allowed;
}

.btn-send:disabled .icon-sm {
  color: var(--text-muted);
}

.btn-stop {
  background: var(--danger) !important;
  box-shadow: 0 4px 16px rgba(255, 71, 87, 0.3);
  animation: stopPulse 1.5s infinite;
}

.btn-stop:hover {
  background: #e63946 !important;
  transform: scale(1.08);
}

@keyframes stopPulse {
  0%, 100% {
    box-shadow: 0 4px 16px rgba(255, 71, 87, 0.3);
  }
  50% {
    box-shadow: 0 4px 24px rgba(255, 71, 87, 0.5);
  }
}

.input-hint {
  font-size: 11px;
  color: var(--text-muted);
  text-align: center;
  margin-top: 8px;
}

/* ===== 网页内全屏模式 ===== */
.fullscreen-backdrop {
  position: fixed;
  inset: 0;
  background: var(--bg-primary);
  z-index: 999;
}

.chat-window.fullscreen-mode {
  position: fixed;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  width: 90vw !important;
  max-width: 1200px;
  height: 85vh !important;
  border-radius: var(--radius-xl) !important;
  z-index: 1000;
}

.chat-window.fullscreen-mode .messages {
  max-height: calc(85vh - 160px);
}

/* ===== 加载动画 ===== */
.typing-indicator {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 8px 4px;
}

.typing-dot {
  width: 8px;
  height: 8px;
  background: var(--accent-primary);
  border-radius: 50%;
  animation: typingBounce 1.4s infinite ease-in-out;
}

.typing-dot:nth-child(1) { animation-delay: 0s; }
.typing-dot:nth-child(2) { animation-delay: 0.2s; }
.typing-dot:nth-child(3) { animation-delay: 0.4s; }

@keyframes typingBounce {
  0%, 60%, 100% {
    transform: translateY(0);
    opacity: 0.4;
  }
  30% {
    transform: translateY(-8px);
    opacity: 1;
  }
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

  .history-drawer {
    width: 100%;
  }

  .floating-ball {
    bottom: 16px;
    right: 16px;
    width: 52px;
    height: 52px;
  }
}
</style>
