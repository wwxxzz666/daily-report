<template>
  <div class="reports">
    <div class="header">
      <h2>报告管理</h2>
      <div class="actions">
        <button class="btn btn-primary" @click="generateDaily" :disabled="generating">
          生成日报
        </button>
        <button class="btn btn-secondary" @click="generateWeekly" :disabled="generating">
          生成周报
        </button>
      </div>
    </div>

    <div v-if="message" class="message" :class="messageType">{{ message }}</div>

    <div v-if="reports && reports.length > 0" class="report-list">
      <div v-for="r in reports" :key="r.filePath" class="report-card">
        <div class="report-icon">{{ r.reportType === 'daily' ? '&#9997;' : '&#128203;' }}</div>
        <div class="report-info">
          <div class="report-name">{{ r.fileName }}</div>
          <div class="report-date">{{ r.generatedAt }}</div>
        </div>
        <div class="report-actions">
          <button class="btn-small" @click="openFile(r.filePath)">打开</button>
          <button class="btn-small" @click="openDir(r.filePath)">文件夹</button>
        </div>
      </div>
    </div>
    <div v-else class="empty">
      暂无报告<br/>
      <span class="empty-sub">点击上方按钮生成你的第一份报告</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import {
  GetReportHistory,
  GenerateDailyReport,
  GenerateWeeklyReport,
  OpenFile,
  OpenReportsDir,
} from '../../wailsjs/go/main/App'

const reports = ref<any[]>([])
const generating = ref(false)
const message = ref('')
const messageType = ref('success')

async function loadReports() {
  try {
    const list = await GetReportHistory()
    reports.value = list || []
  } catch (e) {
    console.error('load reports error:', e)
  }
}

async function generateDaily() {
  generating.value = true
  message.value = ''
  try {
    await GenerateDailyReport()
    message.value = '日报已生成'
    messageType.value = 'success'
    await loadReports()
  } catch (e: any) {
    message.value = '生成失败: ' + (e.message || e)
    messageType.value = 'error'
  } finally {
    generating.value = false
  }
}

async function generateWeekly() {
  generating.value = true
  message.value = ''
  try {
    await GenerateWeeklyReport()
    message.value = '周报已生成'
    messageType.value = 'success'
    await loadReports()
    setTimeout(() => { message.value = '' }, 4000)
  } catch (e: any) {
    message.value = '生成失败: ' + (e.message || e)
    messageType.value = 'error'
  } finally {
    generating.value = false
  }
}

async function openFile(path: string) {
  try {
    await OpenFile(path)
  } catch (e) {
    console.error('open file error:', e)
  }
}

async function openDir(path: string) {
  try {
    await OpenReportsDir()
  } catch (e) {
    console.error('open dir error:', e)
  }
}

onMounted(() => {
  loadReports()
})
</script>

<style scoped>
.reports {
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

.header .actions {
  display: flex;
  gap: 8px;
}

.btn {
  padding: 8px 14px;
  border: none;
  border-radius: 8px;
  font-size: 13px;
  cursor: pointer;
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

.report-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.report-card {
  background: #fff;
  border-radius: 12px;
  padding: 14px;
  display: flex;
  align-items: center;
  gap: 12px;
  box-shadow: 0 1px 3px rgba(0,0,0,0.06);
}

.report-icon {
  font-size: 24px;
}

.report-info {
  flex: 1;
}

.report-name {
  font-size: 14px;
  font-weight: 500;
  color: #333;
}

.report-date {
  font-size: 12px;
  color: #999;
  margin-top: 2px;
}

.report-actions {
  display: flex;
  gap: 6px;
}

.btn-small {
  padding: 5px 12px;
  border: 1px solid #e0e0e0;
  border-radius: 6px;
  background: #fff;
  font-size: 12px;
  cursor: pointer;
  color: #555;
}

.btn-small:hover {
  border-color: #18a058;
  color: #18a058;
}

.empty {
  text-align: center;
  color: #ccc;
  font-size: 14px;
  padding: 40px 0;
  background: #fff;
  border-radius: 12px;
}

.empty-sub {
  font-size: 12px;
  margin-top: 4px;
  display: inline-block;
}
</style>
