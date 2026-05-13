<script setup lang="ts">
definePageMeta({
  middleware: ['auth']
})

useKunSeoMeta({ title: '发布 Galgame' })
</script>

<template>
  <EditGalgameWizard />
</template>
