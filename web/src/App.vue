<template>
  <n-config-provider :theme="darkTheme" :theme-overrides="themeOverrides">
    <n-message-provider>
      <n-dialog-provider>
        <div id="app-root">
          <router-view />
        </div>
      </n-dialog-provider>
    </n-message-provider>
  </n-config-provider>
</template>

<script setup lang="ts">
import { darkTheme } from 'naive-ui'
import type { GlobalThemeOverrides } from 'naive-ui'

const themeOverrides: GlobalThemeOverrides = {
  common: {
    primaryColor: '#6c5ce7',
    primaryColorHover: '#7d6ef0',
    primaryColorPressed: '#5a4bd1',
    primaryColorSuppl: '#6c5ce7',
    borderRadius: '12px',
    bodyColor: '#0f0f1a',
    cardColor: '#1a1a2e',
    modalColor: '#1a1a2e',
    inputColor: '#16162a',
    actionColor: '#16162a',
    hoverColor: '#25254a',
    popoverColor: '#1e1e3a',
    dividerColor: 'rgba(255,255,255,0.08)',
    borderColor: 'rgba(255,255,255,0.06)',
    textColor1: '#e0e0f0',
    textColor2: '#a0a0c0',
    textColor3: 'rgba(255,255,255,0.3)',
    placeholderColor: 'rgba(255,255,255,0.25)',
    railColor: 'rgba(255,255,255,0.08)',
    successColor: '#2ecc71',
    warningColor: '#f39c12',
    errorColor: '#e74c3c',
  },
}
</script>

<style>
* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

body {
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
  background: #0f0f1a;
  color: #e0e0f0;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
}

#app-root {
  min-height: 100vh;
}

::-webkit-scrollbar {
  width: 6px;
  height: 6px;
}

::-webkit-scrollbar-track {
  background: transparent;
}

::-webkit-scrollbar-thumb {
  background: rgba(108, 92, 231, 0.3);
  border-radius: 3px;
}

::-webkit-scrollbar-thumb:hover {
  background: rgba(108, 92, 231, 0.5);
}
</style>
