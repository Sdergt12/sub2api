<template>
  <Transition name="gundam-boot-fade">
    <div v-if="visible" class="gundam-boot-overlay" role="status" aria-live="polite" @click="skip">
      <div class="gundam-boot-grid"></div>
      <div class="gundam-boot-frame">
        <div class="gundam-boot-corners"></div>
        <div class="gundam-boot-unit" aria-hidden="true">
          <svg viewBox="0 0 260 220" class="gundam-boot-unit-svg">
            <path d="M130 18l35 28 13 52 39 27-39 20-11 50H93l-11-50-39-20 39-27 13-52z" />
            <path d="M93 70h74l-16 34h-42z" />
            <path d="M109 122h42l-8 44h-26z" />
            <path d="M76 145l-38 42m146-42l38 42M101 51l-34-24m92 24l34-24" />
          </svg>
        </div>
        <div class="gundam-boot-copy">
          <p class="gundam-boot-kicker">MOBILE SUIT INTERFACE</p>
          <h2>GUNDAM MODE</h2>
          <div class="gundam-boot-status">
            <span>HUD GRID</span>
            <b>ONLINE</b>
          </div>
          <div class="gundam-boot-status">
            <span>RISK PANEL</span>
            <b>SYNC</b>
          </div>
          <div class="gundam-boot-status">
            <span>MECHA THEME</span>
            <b>READY</b>
          </div>
        </div>
        <button class="gundam-boot-skip" type="button" @click.stop="skip">跳过</button>
      </div>
      <div class="gundam-boot-scan"></div>
    </div>
  </Transition>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { useAppStore } from '@/stores/app'

const appStore = useAppStore()
const visible = ref(false)
let timer: ReturnType<typeof setTimeout> | undefined

function prefersReducedMotion(): boolean {
  return typeof window !== 'undefined' && window.matchMedia('(prefers-reduced-motion: reduce)').matches
}

function skip() {
  visible.value = false
  if (timer) clearTimeout(timer)
}

watch(
  () => appStore.gundamBootNonce,
  (nonce) => {
    if (!nonce) return
    visible.value = true
    if (timer) clearTimeout(timer)
    timer = setTimeout(skip, prefersReducedMotion() ? 450 : 1800)
  }
)
</script>
