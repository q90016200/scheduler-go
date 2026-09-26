import type { Directive } from 'vue'

// Liquid glass（參考 kube.io/blog/liquid-glass-css-svg）：依元素實際尺寸產生位移圖與邊緣高光，
// 透過 SVG filter 當 backdrop-filter，讓背景在圓角邊緣折射。
// 只有 Chromium 支援 url() 形式的 backdrop-filter；其他瀏覽器維持 style.css 的 blur 玻璃。
// ponytail: 用 userAgentData 判斷 Chromium（Safari/Firefox 沒有），等它們支援後改成真正的特性偵測
const supported =
  typeof navigator !== 'undefined' &&
  'userAgentData' in navigator &&
  !window.matchMedia('(prefers-reduced-transparency: reduce)').matches

const SVG_NS = 'http://www.w3.org/2000/svg'
const BEZEL = 14 // 折射帶寬度（px）
const SCALE = 28 // feDisplacementMap scale，邊緣最大位移約 SCALE / 2 px
const LIGHT = [-Math.SQRT1_2, -Math.SQRT1_2] // 光源在左上

let defs: SVGDefsElement | undefined
let seq = 0

function getDefs() {
  if (!defs) {
    const svg = document.createElementNS(SVG_NS, 'svg')
    svg.setAttribute('aria-hidden', 'true')
    svg.style.cssText = 'position:absolute;width:0;height:0;overflow:hidden;pointer-events:none'
    defs = document.createElementNS(SVG_NS, 'defs')
    svg.appendChild(defs)
    document.body.appendChild(svg)
  }
  return defs
}

// 圓角矩形的 signed distance 與向外法線；(px, py) 以元素中心為原點
function roundRect(px: number, py: number, hw: number, hh: number, r: number) {
  const qx = Math.abs(px) - hw + r
  const qy = Math.abs(py) - hh + r
  const sx = Math.sign(px) || 1
  const sy = Math.sign(py) || 1
  if (qx > 0 && qy > 0) {
    const len = Math.hypot(qx, qy)
    return { d: r - len, nx: (qx / len) * sx, ny: (qy / len) * sy }
  }
  return qx > qy ? { d: r - qx, nx: sx, ny: 0 } : { d: r - qy, nx: 0, ny: sy }
}

export function buildMaps(w: number, h: number, radius: number) {
  const r = Math.max(Math.min(radius, w / 2, h / 2), 0.5)
  const disp = new ImageData(w, h)
  const spec = new ImageData(w, h)
  const dd = disp.data
  const sd = spec.data
  for (let i = 0; i < dd.length; i += 4) {
    dd[i] = dd[i + 1] = dd[i + 2] = 128
    dd[i + 3] = 255
  }
  const band = BEZEL + r
  for (let y = 0; y < h; y++) {
    const edgeRow = y < band || y >= h - band
    // 中段列只需要處理左右兩側的折射帶
    const ranges = edgeRow ? [[0, w]] : [[0, Math.min(band, w)], [Math.max(w - band, 0), w]]
    for (const [x0, x1] of ranges) {
      for (let x = x0; x < x1; x++) {
        const { d, nx, ny } = roundRect(x + 0.5 - w / 2, y + 0.5 - h / 2, w / 2, h / 2, r)
        if (d < 0 || d >= BEZEL) continue
        const t = 1 - d / BEZEL // 1 在最外緣，0 在折射帶內側
        // ponytail: 用 t² 近似凸面玻璃的折射強度，不做 kube.io 的 Snell 定律預計算；想要更準的彎曲再換
        const m = t * t
        const i = (y * w + x) * 4
        dd[i] = 128 - nx * m * 127
        dd[i + 1] = 128 - ny * m * 127
        const rim = Math.abs(nx * LIGHT[0] + ny * LIGHT[1]) ** 2 * t ** 6
        sd[i] = sd[i + 1] = sd[i + 2] = 255
        sd[i + 3] = rim * 150
      }
    }
  }
  const canvas = document.createElement('canvas')
  canvas.width = w
  canvas.height = h
  const ctx = canvas.getContext('2d')!
  ctx.putImageData(disp, 0, 0)
  const dispUrl = canvas.toDataURL()
  ctx.putImageData(spec, 0, 0)
  return { dispUrl, specUrl: canvas.toDataURL() }
}

interface State {
  id: string
  filter: SVGFilterElement
  ro: ResizeObserver
  timer?: number
  size: string
}
const states = new WeakMap<HTMLElement, State>()

function render(el: HTMLElement, s: State) {
  const w = el.offsetWidth
  const h = el.offsetHeight
  const size = `${w}x${h}`
  if (w < 4 || h < 4 || size === s.size) return
  s.size = size
  const { dispUrl, specUrl } = buildMaps(w, h, parseFloat(getComputedStyle(el).borderTopLeftRadius) || 0)
  s.filter.setAttribute('width', String(w))
  s.filter.setAttribute('height', String(h))
  s.filter.innerHTML = `
    <feGaussianBlur in="SourceGraphic" stdDeviation="3" result="blur"/>
    <feImage href="${dispUrl}" x="0" y="0" width="${w}" height="${h}" preserveAspectRatio="none" result="map"/>
    <feDisplacementMap in="blur" in2="map" scale="${SCALE}" xChannelSelector="R" yChannelSelector="G" result="refracted"/>
    <feColorMatrix in="refracted" type="saturate" values="1.6" result="saturated"/>
    <feImage href="${specUrl}" x="0" y="0" width="${w}" height="${h}" preserveAspectRatio="none" result="specular"/>
    <feComposite in="specular" in2="saturated" operator="over"/>`
  el.style.backdropFilter = `url(#${s.id})`
}

export const vLiquidGlass: Directive<HTMLElement> = {
  mounted(el) {
    if (!supported) return
    const filter = document.createElementNS(SVG_NS, 'filter')
    const id = `liquid-glass-${++seq}`
    filter.id = id
    filter.setAttribute('filterUnits', 'userSpaceOnUse')
    filter.setAttribute('x', '0')
    filter.setAttribute('y', '0')
    filter.setAttribute('color-interpolation-filters', 'sRGB')
    getDefs().appendChild(filter)
    const s: State = { id, filter, size: '', ro: new ResizeObserver(() => {
      clearTimeout(s.timer)
      s.timer = window.setTimeout(() => render(el, s), 150)
    }) }
    states.set(el, s)
    s.ro.observe(el)
    render(el, s)
  },
  unmounted(el) {
    const s = states.get(el)
    if (!s) return
    s.ro.disconnect()
    clearTimeout(s.timer)
    s.filter.remove()
    states.delete(el)
  },
}

declare module 'vue' {
  interface GlobalDirectives {
    vLiquidGlass: typeof vLiquidGlass
  }
}
