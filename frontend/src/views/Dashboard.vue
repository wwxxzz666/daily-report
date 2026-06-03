<template>
  <div class="dashboard">
    <div class="header">
      <h2>日报助手</h2>
      <span class="status" :class="{ running: status?.isRunning }">
        {{ status?.isRunning ? '运行中' : '已暂停' }}
      </span>
    </div>

    <div class="cards-row">
      <div v-if="loading" class="loading">加载中...</div>
      <template v-else>
      <div class="card">
        <div class="card-icon">&#9201;</div>
        <div class="card-info">
          <div class="card-title">工作时间</div>
          <div class="card-value">{{ config?.work_time?.start }} - {{ config?.work_time?.end }}</div>
          <div class="card-sub">{{ status?.countdown || '--' }}</div>
        </div>
      </div>
      <div class="card">
        <div class="card-icon">&#9202;</div>
        <div class="card-info">
          <div class="card-title">今日活动</div>
          <div class="card-value">{{ status?.recordedTime || '0m' }}</div>
          <div class="card-sub">{{ status?.appCount || 0 }} 个应用</div>
        </div>
      </div>
      </template>
    </div>

    <div class="section">
      <h3>快捷操作</h3>
      <div class="actions">
        <button class="btn btn-primary" @click="generateDaily" :disabled="generating">
          {{ generating ? '生成中...' : '生成今日日报' }}
        </button>
        <button class="btn btn-secondary" @click="generateWeekly" :disabled="generating">
          {{ generating ? '生成中...' : '生成本周周报' }}
        </button>
      </div>
      <div v-if="message" class="message" :class="messageType">{{ message }}</div>
    </div>

    <div class="section">
      <h3>今日应用使用</h3>
      <div v-if="summary && summary.length > 0" class="usage-list">
        <div v-for="app in summary" :key="app.processName" class="usage-item">
          <div class="usage-name">{{ app.processName }}</div>
          <div class="usage-bar-bg">
            <div class="usage-bar" :style="{ width: app.percentage + '%' }"></div>
          </div>
          <div class="usage-time">{{ app.duration }}</div>
        </div>
      </div>
      <div v-else class="empty">暂无活动记录</div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import {
  GetConfig,
  GetStatus,
  GetTodaySummary,
  GenerateDailyReport,
  GenerateWeeklyReport,
} from '../../wailsjs/go/main/App'

const config = ref<any>(null)
const status = ref<any>(null)
const summary = ref<any[]>([])
const generating = ref(false)
const loading = ref(true)
const message = ref('')
const messageType = ref('success')

let timer: any = null

function flashMessage(text: string, type: string) {
  message.value = text
  messageType.value = type
  setTimeout(() => { message.value = '' }, 5000)
}

async function refresh() {
  try {
    const [c, s, sum] = await Promise.all([
      GetConfig(),
      GetStatus(),
      GetTodaySummary(),
    ])
    config.value = c
    status.value = s
    if (sum && sum.apps) {
      const maxSec = Math.max(...sum.apps.map((a: any) => a.totalSec), 1)
      summary.value = sum.apps.map((a: any) => ({
        ...a,
        percentage: Math.round((a.totalSec / maxSec) * 100),
      }))
    } else {
      summary.value = []
    }
  } catch (e) {
    console.error('refresh error:', e)
  } finally {
    loading.value = false
  }
}

async function generateDaily() {
  generating.value = true
  message.value = ''
  try {
    const path = await GenerateDailyReport()
    flashMessage('日报已生成: ' + path, 'success')
  } catch (e: any) {
    flashMessage('生成失败: ' + (e.message || e), 'error')
  } finally {
    generating.value = false
  }
}

async function generateWeekly() {
  generating.value = true
  message.value = ''
  try {
    const path = await GenerateWeeklyReport()
    flashMessage('周报已生成: ' + path, 'success')
  } catch (e: any) {
    flashMessage('生成失败: ' + (e.message || e), 'error')
  } finally {
    generating.value = false
  }
}

onMounted(() => {
  refresh()
  timer = setInterval(refresh, 5000)
})

onUnmounted(() => {
  if (timer) clearInterval(timer)
})
</script>

<style scoped>
.dashboard {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.header h2 {
  font-size: 20px;
  font-weight: 600;
  color: #333;
}

.status {
  font-size: 12px;
  padding: 3px 10px;
  border-radius: 12px;
  background: #eee;
  color: #999;
}

.status.running {
  background: #e8f5e9;
  color: #18a058;
}

.cards-row {
  display: flex;
  gap: 12px;
}

.card {
  flex: 1;
  background: #fff;
  border-radius: 12px;
  padding: 14px;
  display: flex;
  align-items: center;
  gap: 10px;
  box-shadow: 0 1px 3px rgba(0,0,0,0.06);
}

.card-icon {
  font-size: 28px;
}

.card-title {
  font-size: 12px;
  color: #999;
}

.card-value {
  font-size: 16px;
  font-weight: 600;
  color: #333;
}

.card-sub {
  font-size: 11px;
  color: #bbb;
}

.section {
  background: #fff;
  border-radius: 12px;
  padding: 16px;
  box-shadow: 0 1px 3px rgba(0,0,0,0.06);
}

.section h3 {
  font-size: 14px;
  font-weight: 600;
  color: #333;
  margin-bottom: 12px;
}

.actions {
  display: flex;
  gap: 10px;
}

.btn {
  flex: 1;
  padding: 10px 16px;
  border: none;
  border-radius: 8px;
  font-size: 14px;
  cursor: pointer;
  transition: opacity 0.2s;
}

.btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-primary {
  background: #18a058;
  color: #fff;
}

.btn-secondary {
  background: #f0f0f0;
  color: #333;
}

.message {
  margin-top: 10px;
  padding: 8px 12px;
  border-radius: 6px;
  font-size: 13px;
}

.message.success {
  background: #e8f5e9;
  color: #18a058;
}

.message.error {
  background: #fde8e8;
  color: #d03050;
}

.usage-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.usage-item {
  display: flex;
  align-items: center;
  gap: 10px;
}

.usage-name {
  width: 80px;
  font-size: 13px;
  color: #555;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.usage-bar-bg {
  flex: 1;
  height: 8px;
  background: #f0f0f0;
  border-radius: 4px;
  overflow: hidden;
}

.usage-bar {
  height: 100%;
  background: #18a058;
  border-radius: 4px;
  transition: width 0.5s ease;
}

.usage-time {
  width: 50px;
  font-size: 12px;
  color: #999;
  text-align: right;
}

.empty {
  text-align: center;
  color: #ccc;
  font-size: 13px;
  padding: 20px 0;
}

.loading {
  text-align: center;
  color: #999;
  font-size: 13px;
  padding: 20px 0;
  background: #fff;
  border-radius: 12px;
}
</style>
