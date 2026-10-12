import type { BotAccount } from '@rayleabot/plugin-ui'

export function botLabel(bot: BotAccount) {
  return bot.nickname ? `${bot.nickname}（${bot.bot_id}）` : bot.bot_id
}
