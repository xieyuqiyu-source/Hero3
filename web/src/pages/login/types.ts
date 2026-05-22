export type Faction = 'wei' | 'shu' | 'wu'

export interface General {
  id: string
  name: string
  title: string
  faction: string
}

export interface FactionData {
  key: Faction
  name: string
  subtitle: string
  color: string
  borderActive: string
  motto: string
  desc: string
  traits: string[]
  generals: General[]
}
