<template>
  <Transition name="gundam-boot-fade">
    <div v-if="visible" class="gundam-boot-overlay" role="status" aria-live="polite" @click="skip">
      <div class="gundam-boot-bg"></div>
      <div class="gundam-boot-grid"></div>
      <div class="gundam-boot-noise"></div>
      <div class="gundam-boot-door gundam-boot-door-left" aria-hidden="true"></div>
      <div class="gundam-boot-door gundam-boot-door-right" aria-hidden="true"></div>
      <div class="gundam-boot-hud" aria-hidden="true">
        <span class="gundam-boot-hud-ring"></span>
        <span class="gundam-boot-hud-axis"></span>
        <span class="gundam-boot-hud-axis gundam-boot-hud-axis-y"></span>
      </div>
      <div class="gundam-boot-frame">
        <div class="gundam-boot-corners"></div>
        <div class="gundam-boot-rails" aria-hidden="true"></div>
        <div class="gundam-boot-tag" aria-hidden="true">HGR-13 / MAINTENANCE DECK</div>
        <div class="gundam-boot-unit" aria-hidden="true">
          <div class="gundam-boot-reticle"></div>
          <div class="gundam-boot-unit-scan"></div>
        </div>
        <div class="gundam-boot-copy">
          <p class="gundam-boot-kicker">MOBILE SUIT MAINTENANCE TERMINAL</p>
          <h2>HANGAR OS</h2>
          <div class="gundam-boot-metrics">
            <div class="gundam-boot-status">
              <span>FRAME LOCK</span>
              <b>SEALED</b>
            </div>
            <div class="gundam-boot-status">
              <span>POWER BUS</span>
              <b>STABLE</b>
            </div>
            <div class="gundam-boot-status">
              <span>RISK CORE</span>
              <b>GUARD</b>
            </div>
            <div class="gundam-boot-status">
              <span>DATA LINK</span>
              <b>SYNC</b>
            </div>
          </div>
          <div class="gundam-boot-bars" aria-hidden="true">
            <i style="--boot-width: 86%"></i>
            <i style="--boot-width: 96%"></i>
            <i style="--boot-width: 72%"></i>
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
    timer = setTimeout(skip, prefersReducedMotion() ? 620 : appStore.gundamBootDurationMs)
  }
)
</script>
