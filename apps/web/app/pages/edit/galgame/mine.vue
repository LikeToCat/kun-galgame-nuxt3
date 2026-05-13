<script setup lang="ts">
definePageMeta({
  middleware: ['auth']
})

useKunSeoMeta({ title: '我的 Galgame 提交' })
</script>

<template>
  <EditGalgameMine />
</template>
