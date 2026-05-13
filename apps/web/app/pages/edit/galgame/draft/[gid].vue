<script setup lang="ts">
definePageMeta({
  middleware: ['auth']
})

useKunSeoMeta({ title: '编辑草稿' })
</script>

<template>
  <EditGalgameDraft />
</template>
