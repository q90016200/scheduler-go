import cronstrue from 'cronstrue'
import 'cronstrue/locales/zh_TW'

// cron 表達式轉中文說明；CRON_TZ= 前綴 cronstrue 不認得，先拆掉再附在後面
export function describeCron(expr: string): string {
  const m = expr.trim().match(/^CRON_TZ=(\S+)\s+(.*)$/)
  const body = m ? m[2] : expr
  try {
    const text = cronstrue.toString(body, { locale: 'zh_TW' })
    return m ? `${text}（${m[1]}）` : text
  } catch {
    return ''
  }
}
