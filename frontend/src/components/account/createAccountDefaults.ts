import type { AccountPlatform, AdminGroup } from '@/types'

export const DEFAULT_OPENAI_ACCOUNT_PRIORITY = 50

export function normalizeOpenAIPlusGroupName(name: string): string {
  return name.replace(/\s+/g, '').toLowerCase()
}

export function resolveDefaultOpenAIPlusGroup(groups: AdminGroup[]): number | null {
  const plusGroup = groups.find((group) => {
    if (group.platform !== 'openai' || group.status !== 'active') {
      return false
    }
    const normalizedName = normalizeOpenAIPlusGroupName(group.name)
    return normalizedName === 'plus组' || normalizedName === 'plusgroup'
  })

  return plusGroup?.id ?? null
}

export function shouldApplyDefaultOpenAIPlusGroup(input: {
  platform: AccountPlatform
  groupIds: number[]
  userTouchedGroups: boolean
}): boolean {
  return input.platform === 'openai' && input.groupIds.length === 0 && !input.userTouchedGroups
}
