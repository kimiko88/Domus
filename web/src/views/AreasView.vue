<script setup lang="ts">
import { onMounted, computed } from 'vue'
import { useAreasStore } from '../stores/areas'
import { useEntitiesStore } from '../stores/entities'
import { Grid, Layers, Sparkles } from '@lucide/vue'

const areasStore = useAreasStore()
const entitiesStore = useEntitiesStore()

onMounted(async () => {
  try {
    await Promise.all([
      areasStore.fetchAreas(),
      entitiesStore.fetchEntities()
    ])
  } catch (err) {
    console.warn('AreasView data fetch warning:', err)
  }
})

function getEntitiesForArea(areaId: string) {
  return entitiesStore.allEntities.filter(
    (e) => (e.attributes as any)?.area_id === areaId || (e as any).area_id === areaId
  )
}
</script>

<template>
  <div class="areas-view">
    <div class="header-row">
      <div>
        <h2 class="section-title">Aree della Casa</h2>
        <p class="section-subtitle">Organizza e controlla i tuoi dispositivi per stanza o zona</p>
      </div>
    </div>

    <!-- Areas Grid -->
    <div v-if="areasStore.sortedAreas.length > 0" class="areas-grid">
      <div
        v-for="area in areasStore.sortedAreas"
        :key="area.id"
        class="area-card glass-card"
      >
        <div class="area-card-top">
          <div class="area-icon-box">
            <Grid :size="24" />
          </div>
          <div class="area-info">
            <h3 class="area-name">{{ area.name }}</h3>
            <span class="area-count">
              {{ getEntitiesForArea(area.id).length }} Dispositivi/Entità
            </span>
          </div>
        </div>

        <div v-if="getEntitiesForArea(area.id).length > 0" class="area-entities-preview">
          <div
            v-for="entity in getEntitiesForArea(area.id).slice(0, 4)"
            :key="entity.id"
            class="entity-pill"
          >
            <span class="pill-dot" :class="{ 'is-on': entity.state === 'on' }" />
            <span class="pill-name">{{ entity.name }}</span>
          </div>
          <span v-if="getEntitiesForArea(area.id).length > 4" class="more-count">
            +{{ getEntitiesForArea(area.id).length - 4 }} altri
          </span>
        </div>
      </div>
    </div>

    <!-- Empty State -->
    <div v-else class="empty-state glass-card">
      <Layers :size="36" class="empty-icon" />
      <h3 class="empty-title">Nessuna area configurata</h3>
      <p class="empty-desc">
        Configura le aree (es. Soggiorno, Cucina, Camera) per raggruppare i tuoi dispositivi intelligenti.
      </p>
    </div>
  </div>
</template>

<style scoped>
.areas-view {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.header-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.section-title {
  font-size: 22px;
  font-weight: 700;
  color: var(--text-main);
  letter-spacing: -0.02em;
}

.section-subtitle {
  font-size: 14px;
  color: var(--text-muted);
  margin-top: 4px;
}

.areas-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: 20px;
}

.area-card {
  padding: 20px;
  display: flex;
  flex-direction: column;
  gap: 16px;
  border-radius: var(--radius-lg);
  border: 1px solid var(--border-subtle);
  background: var(--bg-glass-card);
  transition: all var(--transition-normal);
}

.area-card:hover {
  transform: translateY(-2px);
  border-color: rgba(255, 255, 255, 0.2);
}

.area-card-top {
  display: flex;
  align-items: center;
  gap: 16px;
}

.area-icon-box {
  width: 48px;
  height: 48px;
  border-radius: var(--radius-md);
  background: linear-gradient(135deg, rgba(14, 165, 233, 0.2) 0%, rgba(99, 102, 241, 0.15) 100%);
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--primary);
  flex-shrink: 0;
}

.area-info {
  display: flex;
  flex-direction: column;
}

.area-name {
  font-size: 17px;
  font-weight: 600;
  color: var(--text-main);
}

.area-count {
  font-size: 13px;
  color: var(--text-muted);
  margin-top: 2px;
}

.area-entities-preview {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  padding-top: 12px;
  border-top: 1px solid var(--border-subtle);
}

.entity-pill {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 10px;
  border-radius: var(--radius-full);
  background: var(--bg-surface);
  border: 1px solid var(--border-card);
  font-size: 12px;
  color: var(--text-main);
}

.pill-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background-color: var(--text-dim);
}

.pill-dot.is-on {
  background-color: #fbbf24;
  box-shadow: 0 0 6px rgba(251, 191, 36, 0.6);
}

.pill-name {
  max-width: 120px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.more-count {
  font-size: 12px;
  color: var(--text-dim);
  font-family: var(--font-mono);
}

.empty-state {
  padding: 48px 24px;
  text-align: center;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  max-width: 500px;
  margin: 30px auto;
  border-radius: var(--radius-xl);
}

.empty-icon {
  color: var(--primary);
  margin-bottom: 16px;
  opacity: 0.8;
}

.empty-title {
  font-size: 18px;
  font-weight: 600;
  color: var(--text-main);
}

.empty-desc {
  font-size: 14px;
  color: var(--text-muted);
  margin-top: 8px;
  line-height: 1.5;
}
</style>
