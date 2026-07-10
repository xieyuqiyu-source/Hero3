// 本文件验证军情战报前端展示规则的纯逻辑。
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

import {
  REPORT_SOURCE_CONFIG,
  REPORT_VIEW_CONFIG,
  REPORT_VIEW_TABS,
  buildReportShareURL,
  hasStandardUnitRows,
  hasTraitEntries,
  reportTotalPages,
  shouldRenderSecondarySide,
  shouldShowEmptyReports,
} from '../src/pages/news/reportPresentation.ts'

test('军情页固定展示全部、进攻、防守、增援、侦查、扫荡 Tab', () => {
	assert.deepEqual(REPORT_VIEW_TABS.map((tab) => tab.key), ['all', 'attack', 'defense', 'reinforcement', 'scout', 'sweep'])
})

test('战斗结果弹窗分享前创建公开 token', () => {
	const source = readFileSync(new URL('../src/pages/map/components/BattleResultModal.tsx', import.meta.url), 'utf8')
	assert.match(source, /gameApi\.shareReport/)
	assert.match(source, /buildReportShareURL/)
	assert.doesNotMatch(source, /window\.location\.origin\}\/report\/\$\{report\.id\}/)
})

test('来源和视角标签使用不同颜色配置', () => {
  assert.match(REPORT_VIEW_CONFIG.attack.color, /red/)
  assert.match(REPORT_VIEW_CONFIG.defense.color, /blue/)
  assert.match(REPORT_VIEW_CONFIG.reinforcement.color, /green/)
  assert.match(REPORT_VIEW_CONFIG.scout.color, /yellow/)
  assert.match(REPORT_SOURCE_CONFIG.npc_city.color, /cyan/)
  assert.match(REPORT_SOURCE_CONFIG.player_city.color, /pink/)
  assert.match(REPORT_SOURCE_CONFIG.stronghold.color, /amber/)
  assert.match(REPORT_SOURCE_CONFIG.dungeon.color, /purple/)
})

test('增援战报或隐藏敌方剩余兵力时不渲染下半部分', () => {
  assert.equal(shouldRenderSecondarySide({
    secondarySide: undefined,
    visibility: { showEnemyRemainingUnits: true, showEnemyResources: true, showEnemyGenerals: true, showEnemyCityDefense: true },
  }), false)
  assert.equal(shouldRenderSecondarySide({
    secondarySide: { role: 'defender', units: [], power: 0 },
    visibility: { showEnemyRemainingUnits: false, showEnemyResources: true, showEnemyGenerals: true, showEnemyCityDefense: true },
  }), false)
  assert.equal(shouldRenderSecondarySide({
    secondarySide: { role: 'defender', units: [], power: 0 },
    visibility: { showEnemyRemainingUnits: true, showEnemyResources: true, showEnemyGenerals: true, showEnemyCityDefense: true },
  }), true)
})

test('分享链接优先使用 token，不直接暴露内部战报 ID', () => {
  assert.equal(
    buildReportShareURL('http://localhost:5173', { id: 'report_internal', share: { token: 'token_public' } }),
    'http://localhost:5173/report/token_public',
  )
  assert.equal(
    buildReportShareURL('http://localhost:5173', { id: 'report_internal', detail: { share: { token: 'token_detail' } } }),
    'http://localhost:5173/report/token_detail',
  )
  assert.equal(buildReportShareURL('http://localhost:5173', { id: 'report_internal' }), '')
})

test('空列表和分页状态稳定', () => {
  assert.equal(reportTotalPages(0, 10), 1)
  assert.equal(reportTotalPages(21, 10), 3)
  assert.equal(shouldShowEmptyReports(0, false), true)
  assert.equal(shouldShowEmptyReports(0, true), false)
  assert.equal(shouldShowEmptyReports(1, false), false)
})

test('标准详情支持全兵种固定列和特性触发展示', () => {
  assert.equal(hasStandardUnitRows({
    units: [
      { unitType: 'weiInfantry', unitName: '魏步兵', faction: 'wei', amountBefore: 10, dispatched: 10, lost: 2, survived: 8 },
      { unitType: 'weiCavalry', unitName: '魏骑兵', faction: 'wei', amountBefore: 0, dispatched: 0, lost: 0, survived: 0 },
    ],
  }), true)
  assert.equal(hasTraitEntries({
    traits: [
      { traitId: 'raid', traitName: '奇袭', ownerSide: 'primary', ownerRole: 'attacker', summary: '提高伤害' },
    ],
  }), true)
  assert.equal(hasTraitEntries({ traits: [] }), false)
})
