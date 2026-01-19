<template>
  <div class="grid grid-cols-1 md:grid-cols-3 gap-8 max-w-6xl mx-auto py-16 px-4">
    <div v-for="feature in features" :key="feature.title" 
         class="p-6 rounded-lg border-2 border-wizard-glow bg-slate-900/50 hover:magic-border transition-all duration-300">
      <h3 class="text-2xl font-bold gold-glow mb-4">{{ feature.title }}</h3>
      <p class="text-slate-300 leading-relaxed">{{ feature.description }}</p>
    </div>
  </div>
</template>

<script setup>
const features = [
  {
    title: 'Coach',
    description: 'Enforces structured tasks with Goals, Definition of Done, and Evidence requirements. No more vague loops.'
  },
  {
    title: 'Guard',
    description: 'Hard token and iteration budgets. Real-time command filtering and safety boundaries for every tool call.'
  },
  {
    title: 'Runtime',
    description: 'Episodic execution with rolling summarization. Prevents context collapse in long-running coding sessions.'
  }
]
</script>
