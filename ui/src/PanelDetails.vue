<script setup lang="ts">
interface Skill { name: string; level?: number; active: boolean; description?: string }
export interface Panel { id: string; name: string; skills?: Skill[]; ranks?: Skill[] }
defineProps<{ panel: Panel }>()
</script>
<template>
  <section v-if="panel.skills?.length || panel.ranks?.length" class="business-view">
    <h2>{{ panel.name }} · 技能与解锁效果</h2>
    <details v-for="(skill, index) in panel.skills" :key="`skill-${index}`" class="result-section">
      <summary>{{ skill.name }} · {{ skill.active ? `等级 ${skill.level ?? 0}` : '未解锁' }}</summary>
      <p class="prose muted">{{ skill.description || '官方未提供技能说明。' }}</p>
    </details>
    <details v-for="(rank, index) in panel.ranks" :key="`rank-${index}`" class="result-section">
      <summary>{{ rank.name }} · {{ rank.active ? '已解锁' : '未解锁' }}</summary>
      <p class="prose muted">{{ rank.description || '官方未提供效果说明。' }}</p>
    </details>
  </section>
</template>
