import protossIcon from '../assets/races/protoss.png'
import terranIcon from '../assets/races/terran.png'
import zergIcon from '../assets/races/zerg.png'
import randomIcon from '../assets/races/random.png'

export type RaceId = 'protoss' | 'terran' | 'zerg' | 'random'

export const RACE_META: Record<
  RaceId,
  { label: string; icon: string; statClass: string }
> = {
  protoss: {
    label: 'Protoss',
    icon: protossIcon,
    statClass: 'telemetry__stat--protoss',
  },
  terran: {
    label: 'Terran',
    icon: terranIcon,
    statClass: 'telemetry__stat--terran',
  },
  zerg: {
    label: 'Zerg',
    icon: zergIcon,
    statClass: 'telemetry__stat--zerg',
  },
  random: {
    label: 'Random',
    icon: randomIcon,
    statClass: 'telemetry__stat--random',
  },
}
