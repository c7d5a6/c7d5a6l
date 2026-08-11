import protossIcon from '../assets/races/protoss.png'
import terranIcon from '../assets/races/terran.png'
import zergIcon from '../assets/races/zerg.png'
import randomIcon from '../assets/races/random.png'

export type RaceId = 'protoss' | 'terran' | 'zerg' | 'random'

export const RACE_META: Record<
  RaceId,
  { label: string; icon: string; statClass: string; playerClass: string }
> = {
  protoss: {
    label: 'Protoss',
    icon: protossIcon,
    statClass: 'telemetry__stat--protoss',
    playerClass: 'player--protoss',
  },
  terran: {
    label: 'Terran',
    icon: terranIcon,
    statClass: 'telemetry__stat--terran',
    playerClass: 'player--terran',
  },
  zerg: {
    label: 'Zerg',
    icon: zergIcon,
    statClass: 'telemetry__stat--zerg',
    playerClass: 'player--zerg',
  },
  random: {
    label: 'Random',
    icon: randomIcon,
    statClass: 'telemetry__stat--random',
    playerClass: 'player--random',
  },
}

/** Normalize API / Liquipedia race strings to a known RaceId. */
export function parseRaceId(race: string | null | undefined): RaceId | null {
  if (!race) return null
  const key = race.trim().toLowerCase()
  if (key === 'protoss' || key === 'terran' || key === 'zerg' || key === 'random') return key
  return null
}
