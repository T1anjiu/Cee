<template>
  <n-config-provider :theme="darkTheme" :theme-overrides="themeOverrides">
    <n-global-style />
    <n-message-provider>
      <n-dialog-provider>
        <router-view />
      </n-dialog-provider>
    </n-message-provider>
  </n-config-provider>
</template>

<script setup lang="ts">
import { darkTheme } from 'naive-ui'
import type { GlobalThemeOverrides } from 'naive-ui'

const themeOverrides: GlobalThemeOverrides = {
  common: {
    primaryColor: '#6366f1',
    primaryColorHover: '#818cf8',
    primaryColorPressed: '#4f46e5',
    primaryColorSuppl: '#6366f1',
    borderRadius: '8px',
    fontSize: '14px',
    fontSizeSmall: '12px',
    fontSizeMedium: '14px',
    fontSizeLarge: '16px',
    bodyColor: '#0c0c14',
    cardColor: '#14141f',
    modalColor: '#14141f',
    inputColor: '#1a1a28',
    actionColor: '#1a1a28',
    hoverColor: '#1e1e30',
    popoverColor: '#1a1a28',
    dividerColor: 'rgba(255,255,255,0.06)',
    borderColor: 'rgba(255,255,255,0.06)',
    textColor1: '#e8e8f0',
    textColor2: '#a8a8c0',
    textColor3: 'rgba(255,255,255,0.25)',
    placeholderColor: 'rgba(255,255,255,0.2)',
    railColor: 'rgba(255,255,255,0.06)',
    successColor: '#34d399',
  },
}
</script>

<style>
* { margin: 0; padding: 0; box-sizing: border-box; }
body {
  font-family: -apple-system, BlinkMacSystemFont, 'SF Pro Display', 'Segoe UI', Roboto, 'Helvetica Neue', sans-serif;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
  background: #0c0c14;
}
::-webkit-scrollbar { width: 4px; height: 4px; }
::-webkit-scrollbar-track { background: transparent; }
::-webkit-scrollbar-thumb { background: rgba(255,255,255,0.08); border-radius: 2px; }
::-webkit-scrollbar-thumb:hover { background: rgba(255,255,255,0.15); }
</style>
