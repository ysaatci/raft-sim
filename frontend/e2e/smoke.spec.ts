import { expect, test, type Page } from '@playwright/test'

const leader = (page: Page) => page.getByRole('button', { name: /^Node \d, leader/ })

async function fastForward(page: Page) {
  await page.getByRole('button', { name: '1×', exact: true }).click()
}

test('elects a leader, and elects a new one after the leader crashes', async ({ page }) => {
  await page.goto('/')
  await expect(page.getByRole('button', { name: /^Node \d,/ })).toHaveCount(5)
  await fastForward(page)
  await expect(leader(page)).toHaveCount(1, { timeout: 15_000 })

  const first = (await leader(page).getAttribute('aria-label'))!
  const id = first.match(/^Node (\d)/)![1]
  await leader(page).click()
  await page.getByRole('button', { name: 'Crash', exact: true }).click()
  await expect(page.getByRole('button', { name: new RegExp(`^Node ${id}, crashed`) })).toBeVisible()

  // A different node takes over in a higher term.
  await expect(async () => {
    const label = (await leader(page).getAttribute('aria-label', { timeout: 1000 }))!
    const [, newId, term] = label.match(/^Node (\d), leader, term (\d+)/)!
    expect(newId).not.toBe(id)
    expect(Number(term)).toBeGreaterThan(1)
  }).toPass({ timeout: 15_000 })
  await expect(page.getByLabel('Safety checks: all passing')).toBeVisible()
})

test('plays a guided scenario with narration', async ({ page }) => {
  await page.goto('/')
  await page.getByRole('button', { name: /^Split vote/ }).click()
  const narration = page.getByRole('region', { name: 'Narration' })
  await expect(narration).toContainText('step 1 of 5')
  await narration.getByRole('button', { name: 'Next step' }).click()
  await expect(narration).toContainText('both become candidates for term 1')
  await expect(page.getByRole('button', { name: /^Node \d,/ })).toHaveCount(4)
})

test('a shared link reopens the same moment', async ({ page, context }) => {
  await context.grantPermissions(['clipboard-read', 'clipboard-write'])
  await page.goto('/')
  await fastForward(page)
  await expect(leader(page)).toHaveCount(1, { timeout: 15_000 })
  await page.getByRole('button', { name: 'Pause' }).click()
  const nodes = page.getByRole('button', { name: /^Node \d,/ })
  const before = await nodes.evaluateAll((els) => els.map((e) => e.getAttribute('aria-label')))
  const time = await page.getByLabel('Simulated time').textContent()

  await page.getByRole('button', { name: 'Copy link to this moment' }).click()
  await expect(page).toHaveURL(/#run=/)
  await page.reload()

  await expect(page.getByLabel('Simulated time')).toHaveText(time!)
  expect(await nodes.evaluateAll((els) => els.map((e) => e.getAttribute('aria-label')))).toEqual(before)
  await expect(page.getByRole('button', { name: 'Play' })).toBeVisible() // opens paused
})
