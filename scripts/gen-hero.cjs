#!/usr/bin/env node
/**
 * Generate desktop + mobile hero screenshots for the landing page.
 *
 * Usage:
 *   node scripts/gen-hero.cjs           # mock server must be running on :5199
 *   node scripts/gen-hero.cjs --serve   # auto-start vite dev server
 *
 * Output:
 *   apps/website/src/assets/hero-desktop.png
 *   apps/website/src/assets/hero-mobile.png
 */

const { chromium } = require('playwright')
const { spawn } = require('child_process')
const fs = require('fs')
const path = require('path')

const ROOT = path.resolve(__dirname, '..')
const PORT = 5199
const URL = `http://localhost:${PORT}/?mock`

async function waitForServer(url, timeoutMs = 15000) {
  const start = Date.now()
  while (Date.now() - start < timeoutMs) {
    try {
      const r = await fetch(url)
      if (r.ok) return
    } catch {}
    await new Promise(r => setTimeout(r, 200))
  }
  throw new Error(`Server not ready at ${url} after ${timeoutMs}ms`)
}

async function preparePage(page) {
  await page.goto(URL, { timeout: 5000, waitUntil: 'load' })
  await page.waitForSelector('.session-item', { timeout: 5000 })

  await page.addStyleTag({
    content: `
      *, *::before, *::after {
        transition: none !important;
        animation-duration: 0s !important;
        animation-delay: 0s !important;
        animation-iteration-count: 1 !important;
        caret-color: transparent !important;
      }

      .session-dot-indicator.working,
      .terminal-loading-dot,
      .session-dot.working,
      .main-header-status .session-dot {
        opacity: 1 !important;
        animation: none !important;
        transform: none !important;
        filter: saturate(1.15) brightness(1.12) !important;
      }

      .terminal-shell, .terminal-container, .terminal, .xterm-scrollable-element, .xterm-screen {
        width: 2000px !important;
      }
    `,
  })

  await page.evaluate(() => {
    document.documentElement.classList.add('hero-capture')
  })

  const viewport = page.viewportSize()
  await page.setViewportSize({ width: viewport.width + 1, height: viewport.height })

  await page.waitForTimeout(300)
}

async function takeDesktop(browser) {
  console.log('Taking desktop screenshot...')
  const page = await browser.newPage({
    viewport: { width: 800, height: 650 },
    deviceScaleFactor: 2,
  })

  await preparePage(page)

  const outPath = path.join(ROOT, 'apps/website/src/assets/hero-desktop.png')
  await page.screenshot({ path: outPath })
  await page.close()

  const stat = fs.statSync(outPath)
  console.log(`  → ${path.relative(ROOT, outPath)} (${(stat.size / 1024).toFixed(0)}KB)`)
  return outPath
}

async function takeMobile(browser) {
  console.log('Taking mobile screenshot...')
  const page = await browser.newPage({
    viewport: { width: 390, height: 760 },
    deviceScaleFactor: 2,
    isMobile: true,
    hasTouch: true,
  })

  await preparePage(page)
  await page.waitForSelector('.mobile-bottom-bar', { timeout: 5000 })

  const menuBtn = await page.$('.mobile-bottom-bar button:first-child')
  if (menuBtn) {
    await menuBtn.click()
    await page.waitForTimeout(350)
  }

  const outPath = path.join(ROOT, 'apps/website/src/assets/hero-mobile.png')
  await page.screenshot({ path: outPath })
  await page.close()

  const stat = fs.statSync(outPath)
  console.log(`  → ${path.relative(ROOT, outPath)} (${(stat.size / 1024).toFixed(0)}KB)`)
  return outPath
}

;(async () => {
  const shouldServe = process.argv.includes('--serve')
  let server = null

  try {
    if (shouldServe) {
      console.log('Starting vite dev server...')
      server = spawn('npx', ['vite', '--port', String(PORT)], {
        cwd: path.join(ROOT, 'apps/gmux-web'),
        env: { ...process.env, VITE_MOCK: '1' },
        stdio: 'pipe',
      })
    }

    await waitForServer(URL)
    console.log('Server ready.')

    const browser = await chromium.launch()
    await takeDesktop(browser)
    await takeMobile(browser)
    await browser.close()

    console.log('\n✓ Hero screenshots generated.')
  } finally {
    if (server) server.kill('SIGTERM')
  }
})()
