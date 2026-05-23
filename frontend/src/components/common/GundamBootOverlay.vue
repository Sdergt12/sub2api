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
        <div class="gundam-boot-tag" aria-hidden="true">HGR-07 / MAINTENANCE</div>
        <div class="gundam-boot-unit" aria-hidden="true">
          <svg viewBox="0 0 260 220" class="gundam-boot-unit-svg">
            <path d="M130 16l42 24 13 46 34 20-28 19-10 54-30 18h-42l-30-18-10-54-28-19 34-20 13-46z" />
            <path d="M95 61h70l-10 40-25 16-25-16z" />
            <path d="M85 130h90l-13 41-31 18-33-18z" />
            <path d="M73 92l-42 20m156-20l42 20M92 40l-40-18m116 18l40-18M104 190l-21 22m73-22l21 22" />
            <path d="M113 74h34M111 139h38M128 118v62" />
          </svg>
          <div class="gundam-boot-reticle"></div>
          <div class="gundam-boot-unit-scan"></div>
        </div>
        <div class="gundam-boot-copy">
          <p class="gundam-boot-kicker">MOBILE SUIT MAINTENANCE OS</p>
          <h2>HANGAR BOOT</h2>
          <div class="gundam-boot-metrics">
            <div class="gundam-boot-status">
              <span>ARMOR LOCK</span>
              <b>SEALED</b>
            </div>
            <div class="gundam-boot-status">
              <span>POWER BUS</span>
              <b>GREEN</b>
            </div>
            <div class="gundam-boot-status">
              <span>RISK CORE</span>
              <b>ARMED</b>
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
    timer = setTimeout(skip, prefersReducedMotion() ? 520 : 2100)
  }
)
</script>
