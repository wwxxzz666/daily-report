<template>
  <div class="app-container">
    <div class="content-area">
      <Dashboard v-if="activeTab === 'dashboard'" />
      <Reports v-else-if="activeTab === 'reports'" />
      <Settings v-else-if="activeTab === 'settings'" />
    </div>
    <div class="tab-bar">
      <div
        class="tab-item"
        :class="{ active: activeTab === 'dashboard' }"
        @click="activeTab = 'dashboard'"
      >
        <span class="tab-icon">&#9635;</span>
        <span class="tab-label">仪表盘</span>
      </div>
      <div
        class="tab-item"
        :class="{ active: activeTab === 'reports' }"
        @click="activeTab = 'reports'"
      >
        <span class="tab-icon">&#9997;</span>
        <span class="tab-label">报告</span>
      </div>
      <div
        class="tab-item"
        :class="{ active: activeTab === 'settings' }"
        @click="activeTab = 'settings'"
      >
        <span class="tab-icon">&#9881;</span>
        <span class="tab-label">设置</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import Dashboard from './views/Dashboard.vue'
import Reports from './views/Reports.vue'
import Settings from './views/Settings.vue'

const activeTab = ref('dashboard')
</script>

<style scoped>
.app-container {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: #f5f5f5;
}

.content-area {
  flex: 1;
  overflow-y: auto;
  padding: 16px;
  padding-bottom: 8px;
}

.tab-bar {
  display: flex;
  background: #fff;
  border-top: 1px solid #e8e8e8;
  padding: 6px 0;
  flex-shrink: 0;
}

.tab-item {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 2px;
  padding: 6px 0;
  cursor: pointer;
  color: #999;
  transition: color 0.2s;
  user-select: none;
}

.tab-item.active {
  color: #18a058;
}

.tab-icon {
  font-size: 20px;
  line-height: 1;
}

.tab-label {
  font-size: 11px;
  line-height: 1;
}
</style>
