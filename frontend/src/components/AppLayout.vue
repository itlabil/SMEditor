<script setup>
// Shared page shell: fills the whole viewport with the dark background
// (no white margins outside it) and controls content width. wide=true
// (the project list) stretches edge to edge so a grid can use the full
// screen; the default (project detail, settings) centers content at a
// readable ~1100px.
defineProps({
  wide: { type: Boolean, default: false },
})
</script>

<template>
  <div class="min-h-screen w-full bg-slate-950 text-slate-100">
    <div :class="['mx-auto flex flex-col gap-6 p-8', wide ? 'max-w-none' : 'max-w-[1100px]']">
      <slot />
    </div>
  </div>
</template>
