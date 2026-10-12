export interface BotAccount { source_protocol: string; id: string; nickname?: string }

export function botKey(bot: BotAccount) { return JSON.stringify([bot.source_protocol, bot.id]) }

export function accountBots(bots: BotAccount[]): BotAccount[] {
  const accounts = new Map<string, BotAccount>()
  for (const bot of bots) {
    if (!bot.id) continue
    const key = botKey(bot)
    if (!accounts.has(key) || !accounts.get(key)!.nickname) accounts.set(key, bot)
  }
  return [...accounts.values()]
}

export function botLabel(bots: BotAccount[], protocol: string, id: string) {
  if (!id) return '待选择机器人账号'
  const bot = bots.find(bot => bot.source_protocol === protocol && bot.id === id && bot.nickname)
  return bot?.nickname ? `${bot.nickname}（${id}）` : id
}
