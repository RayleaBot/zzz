<script setup lang="ts">
export interface View { title: string; subtitle?: string; rows?: { label: string; value: string }[]; sections?: { title: string; text?: string; rows?: { label: string; value: string }[] }[]; note?: string }
defineProps<{ value: View }>()
</script>
<template>
  <article class="business-view">
    <header><h2>{{ value.title }}</h2><p v-if="value.subtitle" class="muted">{{ value.subtitle }}</p></header>
    <dl v-if="value.rows?.length" class="facts"><div v-for="(row, index) in value.rows" :key="index"><dt>{{ row.label }}</dt><dd>{{ row.value }}</dd></div></dl>
    <section v-for="(section, index) in value.sections" :key="index" class="result-section"><h3>{{ section.title }}</h3><p v-if="section.text" class="prose">{{ section.text }}</p><dl v-if="section.rows?.length" class="facts"><div v-for="(row, rowIndex) in section.rows" :key="rowIndex"><dt>{{ row.label }}</dt><dd>{{ row.value }}</dd></div></dl></section>
    <p v-if="value.note" class="muted prose">{{ value.note }}</p>
  </article>
</template>
