<template>
  <div class="settings">
    <h2>设置</h2>

    <div class="section">
      <h3>AI 模型配置</h3>
      <div class="form-group">
        <label>服务商</label>
        <select v-model="form.provider" @change="onProviderChange">
          <option v-for="p in providers" :key="p.name" :value="p.name">
            {{ p.label }}
          </option>
        </select>
      </div>
      <div class="form-group">
        <label>API Key</label>
        <input
          type="password"
          v-model="form.apiKey"
          :placeholder="currentProviderNeedsKey ? '请输入 API Key' : '本地模型无需 Key'"
          :disabled="!currentProviderNeedsKey"
        />
      </div>
      <div class="form-group">
        <label>模型</label>
        <input v-model="form.model" :placeholder="currentProviderDefaultModel" />
      </div>
      <div class="hint" v-if="form.provider === 'ollama' || form.provider === 'lmstudio'">
        本地模型，数据完全不出本机
      </div>
      <div class="test-row">
        <button class="btn btn-test" @click="testConnection" :disabled="testing">
          {{ testing ? '测试中...' : '测试连接' }}
        </button>
        <span v-if="testResult" class="test-result" :class="testResultType">
          {{ testResult }}
        </span>
      </div>
    </div>

    <div class="section">
      <h3>工作时间</h3>
      <div class="time-row">
        <div class="form-group">
          <label>上班时间</label>
          <input type="time" v-model="form.workStart" />
        </div>
        <div class="form-group">
          <label>下班时间</label>
          <input type="time" v-model="form.workEnd" />
        </div>
      </div>
      <div class="form-group">
        <label>工作日</label>
        <div class="weekday-row">
          <span
            v-for="(d, i) in weekdays"
            :key="i"
            class="weekday-chip"
            :class="{ active: form.weekdays.includes(i + 1) }"
            @click="toggleWeekday(i + 1)"
          >{{ d }}</span>
        </div>
      </div>
    </div>

    <div class="section">
      <h3>报告设置</h3>
      <div class="form-group">
        <label>周报生成日</label>
        <select v-model.number="form.weeklyDay">
          <option :value="1">周一</option>
          <option :value="2">周二</option>
          <option :value="3">周三</option>
          <option :value="4">周四</option>
          <option :value="5">周五</option>
        </select>
      </div>
    </div>

    <button class="btn btn-save" @click="saveSettings" :disabled="saving">
      {{ saving ? '保存中...' : '保存设置' }}
    </button>
    <div v-if="saveMessage" class="message" :class="saveMessageType">{{ saveMessage }}</div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import {
  GetConfig,
  SaveConfig,
  GetProviders,
  TestLLMConnection,
} from '../../wailsjs/go/main/App'

const weekdays = ['一', '二', '三', '四', '五', '六', '日']

const providers = ref<any[]>([])
const form = ref({
  provider: 'deepseek',
  apiKey: '',
  model: '',
  workStart: '09:00',
  workEnd: '18:00',
  weekdays: [1, 2, 3, 4, 5],
  weeklyDay: 5,
})

const testing = ref(false)
const testResult = ref('')
const testResultType = ref('success')
const saving = ref(false)
const saveMessage = ref('')
const saveMessageType = ref('success')

const currentProviderNeedsKey = computed(() => {
  const p = providers.value.find((p: any) => p.name === form.value.provider)
  return p ? p.needsKey : true
})

const currentProviderDefaultModel = computed(() => {
  const p = providers.value.find((p: any) => p.name === form.value.provider)
  return p ? p.defaultModel : ''
})

function onProviderChange() {
  form.value.model = ''
}

function toggleWeekday(day: number) {
  const idx = form.value.weekdays.indexOf(day)
  if (idx >= 0) {
    form.value.weekdays.splice(idx, 1)
  } else {
    form.value.weekdays.push(day)
    form.value.weekdays.sort()
  }
}

async function loadConfig() {
  try {
    const [cfg, provs] = await Promise.all([GetConfig(), GetProviders()])
    if (cfg) {
      form.value.provider = cfg.LLM?.Provider || 'deepseek'
      form.value.apiKey = cfg.LLM?.APIKey || ''
      form.value.model = cfg.LLM?.Model || ''
      form.value.workStart = cfg.WorkTime?.Start || '09:00'
      form.value.workEnd = cfg.WorkTime?.End || '18:00'
      form.value.weekdays = cfg.WorkTime?.Weekdays || [1, 2, 3, 4, 5]
      form.value.weeklyDay = cfg.Report?.WeeklyDay || 5
    }
    if (provs) {
      providers.value = provs
    }
  } catch (e) {
    console.error('load config error:', e)
  }
}

async function testConnection() {
  testing.value = true
  testResult.value = ''
  try {
    // 先保存当前配置，确保测试用的是最新值
    await SaveConfig({
      llm: {
        provider: form.value.provider,
        api_key: form.value.apiKey,
        model: form.value.model,
      },
      work_time: {
        start: form.value.workStart,
        end: form.value.workEnd,
        weekdays: form.value.weekdays,
      },
      report: {
        weekly_day: form.value.weeklyDay,
      },
    } as any)
    const result = await TestLLMConnection()
    testResult.value = result || '连接成功'
    testResultType.value = 'success'
  } catch (e: any) {
    testResult.value = '连接失败: ' + (e.message || e)
    testResultType.value = 'error'
  } finally {
    testing.value = false
  }
}

async function saveSettings() {
  saving.value = true
  saveMessage.value = ''
  try {
    await SaveConfig({
      llm: {
        provider: form.value.provider,
        api_key: form.value.apiKey,
        model: form.value.model,
      },
      work_time: {
        start: form.value.workStart,
        end: form.value.workEnd,
        weekdays: form.value.weekdays,
      },
      report: {
        weekly_day: form.value.weeklyDay,
      },
    } as any)
    saveMessage.value = '设置已保存'
    saveMessageType.value = 'success'
    setTimeout(() => { saveMessage.value = '' }, 3000)
  } catch (e: any) {
    saveMessage.value = '保存失败: ' + (e.message || e)
    saveMessageType.value = 'error'
  } finally {
    saving.value = false
  }
}

onMounted(() => {
  loadConfig()
})
</script>

<style scoped>
.settings {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

h2 {
  font-size: 20px;
  font-weight: 600;
  color: #333;
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
  margin-bottom: 14px;
}

.form-group {
  margin-bottom: 12px;
}

.form-group label {
  display: block;
  font-size: 12px;
  color: #999;
  margin-bottom: 4px;
}

.form-group input,
.form-group select {
  width: 100%;
  padding: 9px 12px;
  border: 1px solid #e0e0e0;
  border-radius: 8px;
  font-size: 14px;
  outline: none;
  transition: border-color 0.2s;
  background: #fff;
}

.form-group input:focus,
.form-group select:focus {
  border-color: #18a058;
}

.form-group input:disabled {
  background: #f5f5f5;
  color: #ccc;
}

.time-row {
  display: flex;
  gap: 12px;
}

.time-row .form-group {
  flex: 1;
}

.weekday-row {
  display: flex;
  gap: 8px;
}

.weekday-chip {
  width: 36px;
  height: 36px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 13px;
  background: #f0f0f0;
  color: #999;
  cursor: pointer;
  transition: all 0.2s;
  user-select: none;
}

.weekday-chip.active {
  background: #18a058;
  color: #fff;
}

.hint {
  font-size: 12px;
  color: #18a058;
  margin-bottom: 12px;
  padding: 6px 10px;
  background: #e8f5e9;
  border-radius: 6px;
}

.test-row {
  display: flex;
  align-items: center;
  gap: 10px;
}

.test-result {
  font-size: 13px;
}

.test-result.success {
  color: #18a058;
}

.test-result.error {
  color: #d03050;
}

.btn {
  padding: 9px 16px;
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

.btn-test {
  background: #f0f0f0;
  color: #333;
}

.btn-save {
  width: 100%;
  background: #18a058;
  color: #fff;
  padding: 12px;
  font-size: 15px;
  font-weight: 500;
}

.message {
  padding: 8px 12px;
  border-radius: 6px;
  font-size: 13px;
  text-align: center;
}

.message.success {
  background: #e8f5e9;
  color: #18a058;
}

.message.error {
  background: #fde8e8;
  color: #d03050;
}
</style>
