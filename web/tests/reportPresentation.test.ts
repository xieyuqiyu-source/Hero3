// 本文件验证军情战报前端展示规则的纯逻辑。
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

import {
  REPORT_SOURCE_CONFIG,
  REPORT_VIEW_CONFIG,
  REPORT_VIEW_TABS,
  buildLegacyBattleReportDetail,
  buildLegacyReportTraits,
  buildReportShareURL,
  isReportShareToken,
  hasStandardUnitRows,
  hasTraitEntries,
  reportTraitBelongsToSide,
  reportTraitRenderKey,
  reportTotalPages,
  normalizeBattleReportDetail,
  resolveReportTraitDisplaySide,
  shouldRenderSecondarySide,
  shouldShowEmptyReports,
  uniqueReportsById,
} from '../src/pages/news/reportPresentation.ts'
import { GENERAL_TRAITS, TRAIT_TARGET_LABELS, formatTraitOutcomeDetail, formatTraitOutcomeDetails, formatTraitTarget, getTraitMeta } from '../src/utils/traits.ts'

test('军情页固定展示全部、进攻、防守、增援、侦查、扫荡 Tab', () => {
	assert.deepEqual(REPORT_VIEW_TABS.map((tab) => tab.key), ['all', 'attack', 'defense', 'reinforcement', 'scout', 'sweep'])
})

test('战斗结果弹窗分享前创建公开 token', () => {
	const source = readFileSync(new URL('../src/pages/map/components/BattleResultModal.tsx', import.meta.url), 'utf8')
	assert.match(source, /gameApi\.shareReport/)
	assert.match(source, /buildReportShareURL/)
	assert.doesNotMatch(source, /window\.location\.origin\}\/report\/\$\{report\.id\}/)
})

test('NPC 请求失败不会回写玩家状态或打开虚假战报', () => {
	const attackPanel = readFileSync(new URL('../src/pages/map/components/AttackPanel.tsx', import.meta.url), 'utf8')
	const cityCard = readFileSync(new URL('../src/pages/map/components/NpcCityCard.tsx', import.meta.url), 'utf8')
	assert.match(attackPanel, /const result = await gameApi\.attackNpc[\s\S]*patchState\([\s\S]*setBattleReport\(result\.battleReport\)/)
	assert.match(cityCard, /const result = await gameApi\.attackNpc[\s\S]*patchState\([\s\S]*onBattleResult\(result\.battleReport\)/)
	for (const source of [attackPanel, cityCard]) {
		const catchBodies = [...source.matchAll(/catch\s*\{([^}]*)\}/g)].map((match) => match[1])
		assert.equal(catchBodies.length > 0, true)
		assert.equal(catchBodies.some((body) => /patchState|setBattleReport|onBattleResult/.test(body)), false)
	}
})

test('PVP 与 NPC 争用失败都只在成功响应后回写权威状态', () => {
	const npcSource = readFileSync(new URL('../src/pages/map/components/AttackPanel.tsx', import.meta.url), 'utf8')
	const pvpSource = readFileSync(new URL('../src/pages/map/components/WorldMapTab.tsx', import.meta.url), 'utf8')
	const npcHandler = npcSource.match(/const handleDispatch = async \(\) => \{[\s\S]*?\n {2}\}\n\n {2}const handleScout/)
	const pvpHandler = pvpSource.match(/const handleMarch = async \(\) => \{[\s\S]*?\n {2}\}\n\n {2}\/\/ handleReinforce/)
	assert.ok(npcHandler)
	assert.ok(pvpHandler)
	assert.match(npcHandler[0], /await gameApi\.attackNpc[\s\S]*patchState\([\s\S]*setBattleReport/)
	assert.match(pvpHandler[0], /await gameApi\.startPvpAttack[\s\S]*patchState\([\s\S]*setMarches/)
	for (const handler of [npcHandler[0], pvpHandler[0]]) {
		const failurePath = handler.match(/\} catch(?: \(err\))? \{[\s\S]*?\} finally/)
		assert.ok(failurePath)
		assert.doesNotMatch(failurePath[0], /patchState|setBattleReport|setMarches|loadMilitaryView/)
	}
})

test('增援与 NPC 争用失败都不会回写旧兵力或虚假战报', () => {
	const npcSource = readFileSync(new URL('../src/pages/map/components/AttackPanel.tsx', import.meta.url), 'utf8')
	const worldMapSource = readFileSync(new URL('../src/pages/map/components/WorldMapTab.tsx', import.meta.url), 'utf8')
	const npcHandler = npcSource.match(/const handleDispatch = async \(\) => \{[\s\S]*?\n {2}\}\n\n {2}const handleScout/)
	const reinforcementHandler = worldMapSource.match(/const handleReinforce = async \(\) => \{[\s\S]*?\n {2}\}\n\n {2}\/\/ panViewport/)
	assert.ok(npcHandler)
	assert.ok(reinforcementHandler)
	assert.match(npcHandler[0], /await gameApi\.attackNpc[\s\S]*patchState\([\s\S]*setBattleReport/)
	assert.match(reinforcementHandler[0], /await gameApi\.sendReinforcement[\s\S]*patchState\(result\.patch\)[\s\S]*setReinforcements/)
	for (const handler of [npcHandler[0], reinforcementHandler[0]]) {
		const failurePath = handler.match(/\} catch(?: \(err\))? \{[\s\S]*?\} finally/)
		assert.ok(failurePath)
		assert.doesNotMatch(failurePath[0], /patchState|setBattleReport|setReinforcements|loadMilitaryView/)
	}
})

test('NPC 侦查失败不会提交任何玩家状态补丁', () => {
	const sources = [
		readFileSync(new URL('../src/pages/map/components/AttackPanel.tsx', import.meta.url), 'utf8'),
		readFileSync(new URL('../src/pages/map/components/NpcCityCard.tsx', import.meta.url), 'utf8'),
	]
	for (const source of sources) {
		assert.match(source, /const result = await gameApi\.scoutNpc[\s\S]*patchState\([\s\S]*(?:setScoutReport|onScoutResult)\(result\.battleReport\)/)
		const scoutCatch = source.match(/const (?:handleScout|executeAction) = async[\s\S]*?gameApi\.scoutNpc[\s\S]*?catch\s*\{([^}]*)\}/)?.[1] ?? ''
		assert.doesNotMatch(scoutCatch, /patchState|setScoutReport|onScoutResult/)
	}
})

test('NPC 进攻、扫荡和侦查成功都同步留城特性的权威结算状态', () => {
	const sources = [
		readFileSync(new URL('../src/pages/map/components/AttackPanel.tsx', import.meta.url), 'utf8'),
		readFileSync(new URL('../src/pages/map/components/NpcCityCard.tsx', import.meta.url), 'utf8'),
		readFileSync(new URL('../src/pages/map/components/NpcCityTab.tsx', import.meta.url), 'utf8'),
	]
	for (const source of sources) {
		for (const field of ['resources', 'resourceProduction', 'resourceSettledAt', 'generalTraitProgress']) {
			assert.match(source, new RegExp(`${field}: result\\.${field}`))
		}
	}
	const api = readFileSync(new URL('../src/api/game.ts', import.meta.url), 'utf8')
	const types = readFileSync(new URL('../src/types/game.ts', import.meta.url), 'utf8')
	assert.match(api, /attackNpc[\s\S]*resourceProduction: GameState\['resourceProduction'\][\s\S]*generalTraitProgress: Record<string, number>/)
	assert.match(api, /scoutNpc[\s\S]*resourceProduction: GameState\['resourceProduction'\][\s\S]*generalTraitProgress: Record<string, number>/)
	assert.match(types, /interface NpcSweepTask[\s\S]*resourceProduction: ResourceProduction[\s\S]*generalTraitProgress: Record<string, number>/)
})

test('战后返兵区域不把所有减损误写成仁德', () => {
	const source = readFileSync(new URL('../src/pages/map/components/BattleResultModal.tsx', import.meta.url), 'utf8')
	assert.match(source, /战后归队兵力/)
	assert.doesNotMatch(source, /仁德·复活归队/)
})

test('孙策追击在 NPC 战报中明确显示为掠夺战胜利特性和实际损失', () => {
	const trait = getTraitMeta('xiaobawang_zhuiji')
	assert.match(trait.description, /掠夺战胜利后/)
	assert.equal(trait.trigger, '掠夺战结算后')
	assert.equal(formatTraitOutcomeDetail('extraLosses', { weiInfantry: 10 }), '追加损失: weiInfantry +10')
})

test('战前真实伤亡与临时压制使用不同展示语义', () => {
	const options = { faction: 'wei', units: { wei: { weiInfantry: { name: '魏步兵' } } } }
	for (const [traitId, rate, trigger] of [['yibing_touxi', 0.35, '进攻/防守/增援战斗前'], ['shuiyan_qijun', 0.35, '战斗前']] as const) {
		const trait = getTraitMeta(traitId)
		assert.equal(trait.trigger, trigger)
		assert.match(trait.description, /真实伤亡/)
		assert.equal(formatTraitOutcomeDetail('preBattleAffected', { weiInfantry: 350 }, options), '战前真实伤亡: 魏步兵 +350')
		assert.equal(formatTraitOutcomeDetail('effectRate', rate), '设计效果比例: 35%')
	}
	assert.equal(formatTraitOutcomeDetail('maxAffectedRate', 0.35), '设计最大影响比例: 35%')
	assert.match(formatTraitOutcomeDetail('suppressedUnits', { weiInfantry: 20 }), /本场压制兵力/)
	for (const [traitId, rate, expected] of [['zhenhe_quanjun', 0.5, '50%'], ['qimen_dunjia', 0.25, '25%']] as const) {
		const trait = getTraitMeta(traitId)
		assert.equal(trait.trigger, traitId === 'qimen_dunjia' ? '进攻/防守/增援战斗前' : '战斗前')
		assert.match(trait.description, /仅本场.*参战.*战后.*保留/)
		assert.equal(formatTraitOutcomeDetail('effectRate', rate), `设计效果比例: ${expected}`)
		if (traitId !== 'qimen_dunjia') assert.equal(formatTraitOutcomeDetail('maxAffectedRate', rate), `设计最大影响比例: ${expected}`)
	}
	const zhangLiao = getTraitMeta('weizhen_zhenhe')
	assert.equal(zhangLiao.name, '震慑全军')
	assert.equal(zhangLiao.trigger, '主动进攻战斗前')
	assert.match(zhangLiao.description, /35% 概率/)
	assert.match(zhangLiao.description, /25% 兵力溃逃/)
	assert.match(zhangLiao.description, /不计死亡，战后完整返回/)
	assert.equal(formatTraitOutcomeDetail('effectRate', 0.25), '设计效果比例: 25%')
	assert.equal(formatTraitOutcomeDetails({ suppressedUnits: { weiInfantry: 25 }, fledUnits: { weiInfantry: 25 }, returnedUnits: { weiInfantry: 25 } }, options), '本场溃逃兵力: 魏步兵 +25；战后返回兵力: 魏步兵 +25')
	assert.match(getTraitMeta('qimen_dunjia').description, /GM 配置/)
	assert.match(getTraitMeta('qimen_dunjia').description, /25%/)
})

test('内政精营明确为留城被动且前端使用后端当前产量', () => {
	const trait = getTraitMeta('neizheng_jingying')
	assert.equal(trait.trigger, '留城被动生效')
	assert.match(trait.description, /主将留城/)
	assert.match(trait.description, /不作为战斗触发/)
	const source = readFileSync(new URL('../src/hooks/useProjectedResources.ts', import.meta.url), 'utf8')
	assert.match(source, /const \{ resources, resourceProduction \} = state/)
	assert.doesNotMatch(source, /neizheng_jingying|productionBonusRate/)
})

test('魏武号令明确为留城产兵且不伪装成战斗触发', () => {
	const trait = getTraitMeta('weiwu_haoling')
	assert.equal(trait.trigger, '留城持续生效')
	assert.match(trait.description, /留城/)
	assert.match(trait.description, /每分钟自动获得 300 虎卫/)
	assert.match(trait.description, /不设产兵上限/)
	assert.match(trait.description, /离城期间停止/)
	assert.match(trait.description, /后端按真实经过时间权威结算/)
	assert.match(trait.description, /前端只投影显示/)
	assert.match(trait.description, /不作为战斗触发/)
	const storeSource = readFileSync(new URL('../src/store/gameStore.ts', import.meta.url), 'utf8')
	const projectionSource = readFileSync(new URL('../src/utils/guardProjection.ts', import.meta.url), 'utf8')
	assert.match(storeSource, /let militaryRequestVersion = 0/)
	assert.match(storeSource, /const requestVersion = \+\+militaryRequestVersion/)
	assert.match(storeSource, /requestVersion !== militaryRequestVersion/)
	assert.doesNotMatch(storeSource, /guardPerMinute|calculateGuardProduction/)
	assert.match(projectionSource, /guardPerMinute/)
	assert.match(projectionSource, /isCaoCaoAtHome/)
	assert.doesNotMatch(projectionSource, /patchState|setState/)
})

test('天神下凡明确增加武将最终武力而不是战斗触发', () => {
	const trait = getTraitMeta('tianshen_xiafan')
	assert.equal(trait.trigger, '被动属性')
	assert.match(trait.description, /20 点武力/)
	assert.match(trait.description, /每点武力转化为 2% 部队攻击/)
})

test('王佐之才明确为留城征兵减耗且不伪装成战斗触发', () => {
	const trait = getTraitMeta('wangzuo_zhicai')
	assert.equal(trait.trigger, '留城征兵消耗时')
	assert.match(trait.description, /留城/)
	assert.match(trait.description, /5%/)
	assert.match(trait.description, /离城失效/)
	assert.match(trait.description, /不作为战斗触发/)
})

test('神鬼之才和内政精营明确永久属性、留城条件与非战斗边界', () => {
	const passive = getTraitMeta('shengui_zhicai')
	const production = getTraitMeta('neizheng_jingying')
	assert.equal(passive.trigger, '永久被动')
	assert.match(passive.description, /10 点内政/)
	assert.match(passive.description, /10 点智谋/)
	assert.match(passive.description, /最终四维/)
	assert.match(passive.description, /不作为战斗触发/)
	assert.equal(production.trigger, '留城被动生效')
	assert.match(production.description, /5%/)
	assert.match(production.description, /离城失效/)
	assert.match(production.description, /不作为战斗触发/)
})

test('征兵弹窗同步阻止快速重复提交并立即采用后端权威结果', () => {
	const source = readFileSync(new URL('../src/pages/military/components/RecruitModal.tsx', import.meta.url), 'utf8')
	assert.match(source, /const recruitingRef = useRef\(false\)/)
	assert.match(source, /recruitingRef\.current = true[\s\S]*await gameApi\.recruit/)
	assert.match(source, /patchMilitaryAction\(result\)[\s\S]*handleClose\(true\)/)
	assert.doesNotMatch(source, /setTimeout\(\(\) => patchMilitaryAction\(result\)/)
})

test('征兵成功同步资源结算时点和特性进度且失败不污染权威状态', () => {
	const modalSource = readFileSync(new URL('../src/pages/military/components/RecruitModal.tsx', import.meta.url), 'utf8')
	const storeSource = readFileSync(new URL('../src/store/gameStore.ts', import.meta.url), 'utf8')
	assert.match(storeSource, /patchMilitaryAction:[\s\S]*resourceProduction: result\.resourceProduction[\s\S]*resourceSettledAt: result\.resourceSettledAt[\s\S]*generalTraitProgress: result\.generalTraitProgress/)
	assert.match(modalSource, /const result = await gameApi\.recruit[\s\S]*patchMilitaryAction\(result\)/)
	const catchBody = modalSource.match(/const handleRecruit = async \(\) => \{[\s\S]*?catch\s*\{([\s\S]*?)\n\s*\}/)?.[1] ?? ''
	assert.doesNotMatch(catchBody, /patchMilitaryAction|patchState/)
})

test('甄宓两项特性展示当前概率、攻防比例和战前方向', () => {
	const trait = getTraitMeta('meiren')
	const reduction = getTraitMeta('meihuo_raozhen')
	assert.equal(trait.name, '美人心计')
	assert.equal(trait.trigger, '主动进攻战斗前')
	assert.match(trait.description, /50%/)
	assert.match(trait.description, /全军攻击提升 25%/)
	assert.equal(reduction.trigger, '主动进攻战斗前')
	assert.match(reduction.description, /50%/)
	assert.match(reduction.description, /防御降低 25%/)
})

test('甄宓两项特性可同时展示实际攻防变化且不再产生俘虏结果', () => {
	const detail = {
		traits: [
			{ traitId: 'meiren', traitName: '美人心计', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'zhenmi', detail: { attackBonusRate: 0.25, attackModifiedUnits: { huWei: 3 }, triggerChance: 0.5 } },
			{ traitId: 'meihuo_raozhen', traitName: '魅惑扰阵', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'zhenmi', detail: { enemyDefenseReductionRate: 0.25, infantryDefenseModifiedUnits: { greedyWolf: -2 }, cavalryDefenseModifiedUnits: { greedyWolf: -2 }, triggerChance: 0.5 } },
		],
	}
	assert.equal(hasTraitEntries(detail), true)
	assert.deepEqual(detail.traits.map((item) => resolveReportTraitDisplaySide({ sourceType: 'player_city', viewType: 'attack' }, item)), ['primary', 'primary'])
	assert.equal(formatTraitOutcomeDetail('attackBonusRate', detail.traits[0].detail.attackBonusRate), '设计攻击加成: 25%')
	assert.equal(formatTraitOutcomeDetail('attackModifiedUnits', detail.traits[0].detail.attackModifiedUnits), '实际攻击修正: 虎卫 +3')
	assert.equal(formatTraitOutcomeDetail('enemyDefenseReductionRate', detail.traits[1].detail.enemyDefenseReductionRate), '设计敌方防御降低: 25%')
	assert.equal(formatTraitOutcomeDetail('infantryDefenseModifiedUnits', detail.traits[1].detail.infantryDefenseModifiedUnits), '实际步防修正: 贪狼营 -2')
	assert.equal(formatTraitOutcomeDetail('triggerChance', 0.5), '触发概率: 50%')
	assert.equal(JSON.stringify(detail).includes('captured'), false)
})

test('火攻按正式配置必定触发且仁德为永久属性被动', () => {
	const fire = getTraitMeta('huogong')
	const rende = getTraitMeta('rende')
	assert.equal(fire.trigger, '主动进攻战斗结算后')
	assert.match(fire.description, /必定触发/)
	assert.match(fire.description, /25%/)
	assert.match(fire.description, /各兵种战前人数尝试追加 25%/)
	assert.equal(formatTraitOutcomeDetail('targetExtraLosses', { weiInfantry: 25 }, { faction: 'wei', units: { wei: { weiInfantry: { name: '魏步兵' } } } }), '目标兵种追加损失: 魏步兵 +25')
	assert.equal(rende.trigger, '永久被动')
	assert.match(rende.description, /固定增加 10 点内政/)
	assert.match(rende.description, /12 点统率/)
	assert.doesNotMatch(rende.description, /触发概率|复活/)
})

test('周瑜双特性按后端时间线区分战前加攻和战后火攻', () => {
	const detail = {
		traits: [
			{
				traitId: 'meizhoulang_junlue', traitName: '美周郎军略', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'zhouyu',
				detail: { attackBonusRate: 0.05, attackModifiedUnits: { shadowGuard: 1 } },
			},
			{
				traitId: 'huogong', traitName: '火烧赤壁', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'zhouyu',
				detail: { damagePercent: 0.25, extraDamage: 250, targetExtraLosses: { greedyWolf: 250 }, triggerChance: 1 },
			},
		],
	}
	assert.equal(hasTraitEntries(detail), true)
	assert.deepEqual(detail.traits.map((trait) => trait.traitId), ['meizhoulang_junlue', 'huogong'])
	assert.deepEqual(detail.traits.map((trait) => getTraitMeta(trait.traitId).trigger), ['主动进攻战斗前', '主动进攻战斗结算后'])
	assert.deepEqual(detail.traits.map((trait) => resolveReportTraitDisplaySide({ sourceType: 'player_city', viewType: 'attack' }, trait)), ['primary', 'primary'])
	assert.equal(formatTraitOutcomeDetail('attackModifiedUnits', detail.traits[0].detail.attackModifiedUnits), '实际攻击修正: 影卫 +1')
	assert.equal(formatTraitOutcomeDetail('targetExtraLosses', detail.traits[1].detail.targetExtraLosses), '目标兵种追加损失: 贪狼营 +250')
	assert.equal(formatTraitOutcomeDetail('extraDamage', detail.traits[1].detail.extraDamage), '额外伤害: 250')
})

test('甄宓 NPC 与 PVP 双特性同时展示真实攻防变化和最终兵力', () => {
	const detail = {
		secondarySide: {
			units: [{ unitType: 'greedyWolf', unitName: '贪狼营', amountBefore: 1000, dispatched: 1000, lost: 800, survived: 0 }],
		},
		traits: [
			{
				traitId: 'meiren', traitName: '美人心计', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'zhenmi',
				detail: { attackBonusRate: 0.25, attackModifiedUnits: { huWei: 3 }, triggerChance: 0.5 },
			},
			{
				traitId: 'meihuo_raozhen', traitName: '魅惑扰阵', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'zhenmi',
				detail: { enemyDefenseReductionRate: 0.25, infantryDefenseModifiedUnits: { greedyWolf: -2 }, cavalryDefenseModifiedUnits: { greedyWolf: -2 }, triggerChance: 0.5 },
			},
		],
	}
	assert.equal(hasTraitEntries(detail), true)
	assert.deepEqual(detail.secondarySide.units[0], { unitType: 'greedyWolf', unitName: '贪狼营', amountBefore: 1000, dispatched: 1000, lost: 800, survived: 0 })
	assert.deepEqual(detail.traits.map((trait) => getTraitMeta(trait.traitId).trigger), ['主动进攻战斗前', '主动进攻战斗前'])
	for (const sourceType of ['npc_city', 'player_city'] as const) {
		assert.deepEqual(detail.traits.map((trait) => resolveReportTraitDisplaySide({ sourceType, viewType: 'attack' }, trait)), ['primary', 'primary'])
	}
	assert.equal(formatTraitOutcomeDetail('attackModifiedUnits', detail.traits[0].detail.attackModifiedUnits), '实际攻击修正: 虎卫 +3')
	assert.equal(formatTraitOutcomeDetail('enemyDefenseReductionRate', detail.traits[1].detail.enemyDefenseReductionRate), '设计敌方防御降低: 25%')
	assert.equal(formatTraitOutcomeDetail('infantryDefenseModifiedUnits', detail.traits[1].detail.infantryDefenseModifiedUnits), '实际步防修正: 贪狼营 -2')
	assert.equal(formatTraitOutcomeDetail('cavalryDefenseModifiedUnits', detail.traits[1].detail.cavalryDefenseModifiedUnits), '实际骑防修正: 贪狼营 -2')
})

test('全部正式随机战斗特性展示准确触发概率', () => {
	const expected = {
		meiren: '50%', meihuo_raozhen: '50%',
		yibing_touxi: '35%', huchi_chongzhen: '50%', sizhandaodi: '60%', weizhen_zhenhe: '35%', weizhen_xiaoyao: '60%',
		shuiyan_qijun: '35%', zhenhe_quanjun: '50%', longdan_jiuyuan: '35%', xiliang_tuji: '35%',
		baibu_chuanyang: '35%', qibing_raohou: '35%', jiangdong_gushou: '50%', xiaobawang_zhuiji: '35%',
		huoshao_lianying: '35%', kurouji: '35%',
	} as const
	for (const [traitId, chance] of Object.entries(expected)) {
		assert.match(getTraitMeta(traitId).description, new RegExp(`有 ${chance} 概率`), traitId)
	}
})

test('兵种限定特性展示真实目标和实际修正', () => {
	assert.match(getTraitMeta('weiwu_tongyu').description, /全军防御提升 15%/)
	assert.match(getTraitMeta('weiwu_tongyu').description, /守城或作为援军/)
	assert.match(getTraitMeta('weiwu_tongyu').description, /主动进攻无效/)
	assert.equal(getTraitMeta('weiwu_tongyu').trigger, '守城/增援战斗前')
	assert.equal(formatTraitTarget('overlordRider'), '霸王骑')
	assert.match(formatTraitOutcomeDetail('modifiedUnits', { overlordRider: 50 }), /实际攻防修正/)
	assert.equal(formatTraitOutcomeDetail('attackModifiedUnits', { overlordRider: 50 }), '实际攻击修正: 霸王骑 +50')
	assert.equal(formatTraitOutcomeDetail('infantryDefenseModifiedUnits', { huWei: 1 }), '实际步防修正: 虎卫 +1')
	assert.equal(formatTraitOutcomeDetail('cavalryDefenseModifiedUnits', { huWei: 1 }), '实际骑防修正: 虎卫 +1')
	assert.equal(formatTraitOutcomeDetail('targetExtraLosses', { overlordRider: 12 }), '目标兵种追加损失: 霸王骑 +12')
	assert.match(getTraitMeta('gushou_hanzhong').description, /所带部队的步兵、骑兵防御各增加 20 点/)
})

test('后端正式特性结果字段全部使用玩家可读中文标签', () => {
	const fields = {
		triggerCount: [2, '触发场次'], triggerChance: [1, '触发概率'], effectRate: [0.1, '设计效果比例'],
		captureRate: [0.2, '俘虏比例'], captureMax: [1000, '设计单兵种俘虏上限'], maxAffectedRate: [0.35, '设计最大影响比例'],
		attackBonusRate: [0.1, '设计攻击加成'], unitAttackFlat: [50, '设计单位攻击增加'], attackReductionRate: [0.1, '设计攻击降低'],
		enemyDefenseReductionRate: [0.2, '设计敌方防御降低'], defenseBonusRate: [0.5, '设计防御加成'], generalDefenseFlat: [20, '设计全军防御增加'],
		lossReductionRate: [0.2, '设计减损比例'], maxReviveCount: [10000, '设计复活上限'], maxReturnCount: [10000, '设计返还上限'],
		plunderBonusRate: [-0.2, '设计掠夺修正'], damagePercent: [0.25, '设计伤害比例'],
		preBattleAffected: [{ wuInfantry: 10 }, '战前真实伤亡'], suppressedUnits: [{ wuInfantry: 10 }, '本场压制兵力'],
		capturedUnits: [{ wuInfantry: 10 }, '俘虏归队'], capturedToGarrison: [{ wuInfantry: 10 }, '俘虏驻防'], totalCaptured: [10, '俘虏总数'],
		modifiedUnits: [{ wuInfantry: 1 }, '实际攻防修正'], attackModifiedUnits: [{ wuInfantry: 1 }, '实际攻击修正'],
		infantryDefenseModifiedUnits: [{ wuInfantry: 1 }, '实际步防修正'], cavalryDefenseModifiedUnits: [{ wuInfantry: 1 }, '实际骑防修正'],
		extraLosses: [{ wuInfantry: 10 }, '追加损失'], targetExtraLosses: [{ wuInfantry: 10 }, '目标兵种追加损失'], extraDamage: [10, '额外伤害'],
		reducedLosses: [{ wuInfantry: 10 }, '减少损失'], revivedUnits: [{ wuInfantry: 10 }, '复活兵力'], totalRevived: [10, '复活总数'],
		returnedUnits: [{ wuInfantry: 10 }, '返还兵力'], disabledTraits: [{ disabledTraitCount: 1 }, '压制特性'],
		disableTraitCount: [1, '设计压制特性数'], disabledTraitCount: [1, '实际压制特性数'], totalSuppressed: [10, '震慑兵力'],
		plunderDelta: [{ wood: -10 }, '掠夺资源修正'],
	} as const
	for (const [key, [value, label]] of Object.entries(fields)) {
		const text = formatTraitOutcomeDetail(key, value)
		assert.equal(text.startsWith(`${label}:`), true, `${key} 仍显示为非正式标签: ${text}`)
		assert.equal(text.startsWith(`${key}:`), false, `${key} 不能直接暴露给玩家`)
	}
})

test('正式兵种战报中文兜底与后端配置逐项一致', () => {
	for (const faction of ['wei', 'shu', 'wu']) {
		const units = JSON.parse(readFileSync(new URL(`../../go/config/units/${faction}.json`, import.meta.url), 'utf8')) as Record<string, { name: string }>
		for (const [unitType, unit] of Object.entries(units)) {
			assert.equal(TRAIT_TARGET_LABELS[unitType], unit.name, `${faction}.${unitType}`)
		}
	}
})

test('掠夺特性区分设计比例、实际资源变化和错误场景', () => {
	const ganning = getTraitMeta('jinfan_jielue')
	const sunquan = getTraitMeta('jiangdong_haoling')
	assert.match(ganning.description, /掠夺收益提升 20%/)
	assert.match(ganning.description, /普通进攻或掠夺战败时无效/)
	assert.match(sunquan.description, /掠夺收益降低 20%/)
	assert.match(sunquan.description, /普通进攻或防守获胜时无效/)
	assert.equal(formatTraitOutcomeDetail('plunderBonusRate', 0.2), '设计掠夺修正: 20%')
	assert.equal(formatTraitOutcomeDetail('plunderBonusRate', -0.2), '设计掠夺修正: -20%')
	assert.equal(formatTraitOutcomeDetail('plunderDelta', { wood: 60, stone: 20 }), '掠夺资源修正: 木材 +60、石料 +20')
	assert.equal(formatTraitOutcomeDetail('plunderDelta', { food: -40 }), '掠夺资源修正: 粮食 -40')
	const outcomes = [
		{ traitId: 'jinfan_jielue', ownerSide: 'primary', detail: { plunderDelta: { wood: 60 } } },
		{ traitId: 'jiangdong_haoling', ownerSide: 'secondary', detail: { plunderDelta: { wood: -60 } } },
	]
	assert.equal(new Set(outcomes.map((outcome) => `${outcome.ownerSide}:${outcome.traitId}`)).size, 2)
})

test('并发结算异常返回同一掠夺战报时列表只保留一份权威记录', () => {
	const reports = [
		{ id: 'plunder-r1', rewards: { wood: 312 } },
		{ id: 'plunder-r1', rewards: { wood: 624 } },
	]
	assert.deepEqual(uniqueReportsById(reports), [{ id: 'plunder-r1', rewards: { wood: 312 } }])
})

test('防守主将特性不会因同将领 ID 复制到援军区', () => {
	const detail = { sourceType: 'player_city', viewType: 'attack' }
	const trait = { ownerSide: 'secondary', ownerRole: 'defender', generalId: 'sunquan' }
	const defender = { role: 'defender', generals: [{ id: 'sunquan' }] }
	const reinforcement = { role: 'reinforcement', generals: [{ id: 'sunquan' }] }
	assert.equal(reportTraitBelongsToSide(detail, trait, defender, 'secondary'), true)
	assert.equal(reportTraitBelongsToSide(detail, trait, reinforcement, 'reinforcement'), false)
})

test('多名玩家同将领援军特性按玩家归属且不复制到其他援军区', () => {
	const detail = { sourceType: 'player_city', viewType: 'attack' }
	const firstSide = { role: 'reinforcement', playerId: 'helper_a', generals: [{ id: 'zhaoyun' }] }
	const secondSide = { role: 'reinforcement', playerId: 'helper_b', generals: [{ id: 'zhaoyun' }] }
	const firstTrait = { ownerSide: 'reinforcement', ownerRole: 'reinforcement', ownerPlayerId: 'helper_a', generalId: 'zhaoyun' }
	const secondTrait = { ownerSide: 'reinforcement', ownerRole: 'reinforcement', ownerPlayerId: 'helper_b', generalId: 'zhaoyun' }
	assert.equal(reportTraitBelongsToSide(detail, firstTrait, firstSide, 'reinforcement'), true)
	assert.equal(reportTraitBelongsToSide(detail, firstTrait, secondSide, 'reinforcement'), false)
	assert.equal(reportTraitBelongsToSide(detail, secondTrait, firstSide, 'reinforcement'), false)
	assert.equal(reportTraitBelongsToSide(detail, secondTrait, secondSide, 'reinforcement'), true)
	const legacyTrait = { ownerSide: 'reinforcement', ownerRole: 'reinforcement', generalId: 'zhaoyun' }
	assert.equal(reportTraitBelongsToSide(detail, legacyTrait, firstSide, 'reinforcement'), true)
	assert.equal(reportTraitBelongsToSide(detail, { ...firstTrait, generalId: 'legacy_wrong_general' }, firstSide, 'reinforcement'), true)
	const firstKey = reportTraitRenderKey({ traitId: 'longdan_jiuyuan', ...firstTrait }, 0)
	const secondKey = reportTraitRenderKey({ traitId: 'longdan_jiuyuan', ...secondTrait }, 1)
	assert.notEqual(firstKey, secondKey)
	assert.match(firstKey, /helper_a/)
	assert.match(secondKey, /helper_b/)
	const panelSource = readFileSync(new URL('../src/pages/news/components/report-detail/ReportTraitPanel.tsx', import.meta.url), 'utf8')
	assert.match(panelSource, /key=\{reportTraitRenderKey\(trait, index\)\}/)
})

test('标准军情详情汇总展示特性的设计参数和实际结算数值', () => {
	const options = { faction: 'shu', units: { shu: { greedyWolf: { name: '贪狼营' } } } }
	assert.equal(
		formatTraitOutcomeDetails({ lossReductionRate: 0.2, reducedLosses: { greedyWolf: 20 } }, options),
		'设计减损比例: 20%；减少损失: 贪狼营 +20',
	)
	assert.equal(formatTraitOutcomeDetails({ disabledTraitCount: 0 }), '实际压制特性数: 0')
	assert.equal(formatTraitOutcomeDetails(), '')

	const detailSource = readFileSync(new URL('../src/pages/news/components/BattleReportDetail.tsx', import.meta.url), 'utf8')
	const panelSource = readFileSync(new URL('../src/pages/news/components/report-detail/ReportTraitPanel.tsx', import.meta.url), 'utf8')
	assert.match(detailSource, /formatTraitOutcomeDetails\(trait\.detail/)
	assert.match(detailSource, /sideTriggeredEffectText\(detail, side, 'reinforcement', unitsConfig\)/)
	assert.match(panelSource, /formatTraitOutcomeDetails\(trait\.detail, formatOptions\)/)
})

test('历史战报恢复同将领多援军的真实特性归属、数值和最终存活', () => {
	const report = {
		id: 'legacy-double-reinforcement', playerId: 'attacker', ownerPlayerId: 'attacker', viewType: 'attack', sourceType: 'player_city',
		battleType: 'attack', winnerSide: 'attacker', ownerSide: 'attacker', ownerOutcome: 'victory', title: '历史双援军',
		playerFaction: 'wei', playerName: '进攻方', targetId: 'defender', targetName: '防守城', type: 'attack', result: 'attacker_victory',
		playerPower: 4000, enemyPower: 3010, dispatchedUnits: { huWei: 100 }, lostUnits: { huWei: 100 }, survivedUnits: { huWei: 10 },
		defenderFaction: 'shu', defenderUnits: { greedyWolf: 1 }, defenderLostUnits: { greedyWolf: 1 }, defenderRevealed: true,
		defenderResources: {}, rewards: { wood: 100 }, drops: [{ type: 'item', itemId: 'token', name: '征战令', amount: 2 }],
		overflowCityGold: 5, generalExpGained: 30, generalLevelBefore: 1, generalLevelAfter: 2, overflow: { wood: 3 }, read: true, createdAt: '2026-07-19T12:00:00Z',
		pvpReinforcements: [
			{ reinforcementId: 'legacy_a', fromPlayerId: 'helper_a', fromPlayerName: '常山甲', faction: 'shu', troops: { greedyWolf: 100 }, generals: [{ id: 'zhaoyun', name: '赵云', level: 1 }] },
			{ reinforcementId: 'legacy_b', fromPlayerId: 'helper_b', fromPlayerName: '常山乙', faction: 'shu', troops: { greedyWolf: 200 }, generals: [{ id: 'zhaoyun', name: '赵云', level: 1 }] },
		],
		pvpReinforcementLosses: { legacy_a: { greedyWolf: 80 }, legacy_b: { greedyWolf: 160 } },
		traitTriggered: ['longdan_jiuyuan', 'longdan_jiuyuan::reinforcement::zhaoyun'],
		traitOutcomes: {
			longdan_jiuyuan: {
				traitId: 'longdan_jiuyuan', name: '龙胆救援', ownerSide: 'reinforcement', ownerPlayerId: 'helper_a', ownerGeneralId: 'zhaoyun',
				detail: { lossReductionRate: 0.2, reducedLosses: { greedyWolf: 20 } },
			},
			'longdan_jiuyuan::reinforcement::zhaoyun': {
				traitId: 'longdan_jiuyuan', name: '龙胆救援', ownerSide: 'reinforcement', ownerPlayerId: 'helper_b', ownerGeneralId: 'zhaoyun',
				detail: { lossReductionRate: 0.2, reducedLosses: { greedyWolf: 40 } },
			},
		},
	}

	const traits = buildLegacyReportTraits(report)
	assert.deepEqual(traits.map((trait) => trait.traitId), ['longdan_jiuyuan', 'longdan_jiuyuan'])
	assert.deepEqual(traits.map((trait) => trait.ownerPlayerId), ['helper_a', 'helper_b'])
	assert.deepEqual(traits.map((trait) => trait.ownerSide), ['reinforcement', 'reinforcement'])
	assert.deepEqual(traits.map((trait) => trait.ownerRole), ['reinforcement', 'reinforcement'])
	assert.deepEqual(traits.map((trait) => trait.generalName), ['赵云', '赵云'])
	assert.deepEqual(traits.map((trait) => formatTraitOutcomeDetails(trait.detail)), [
		'设计减损比例: 20%；减少损失: 贪狼营 +20',
		'设计减损比例: 20%；减少损失: 贪狼营 +40',
	])

	const detail = buildLegacyBattleReportDetail(report)
	assert.equal(detail.primarySide.units[0].survived, 10)
	assert.equal(detail.traits?.length, 2)
	assert.equal(detail.extra?.pvp && typeof detail.extra.pvp === 'object' ? detail.extra.pvp.reinforcements.length : 0, 2)
	assert.equal(reportTraitBelongsToSide(detail, traits[0], { role: 'reinforcement', playerId: 'helper_a', generals: [{ id: 'zhaoyun' }] }, 'reinforcement'), true)
	assert.equal(reportTraitBelongsToSide(detail, traits[0], { role: 'reinforcement', playerId: 'helper_b', generals: [{ id: 'zhaoyun' }] }, 'reinforcement'), false)

	const partialDetailReport = { ...report, detail: { ...detail, traits: [], rewards: {}, extra: { pvp: { wall: { level: 10 } } } } }
	const normalizedPartial = normalizeBattleReportDetail(partialDetailReport)
	assert.equal(normalizedPartial.traits?.length, 2)
	assert.deepEqual(normalizedPartial.traits?.map((trait) => trait.ownerPlayerId), ['helper_a', 'helper_b'])
	assert.equal(normalizedPartial.extra?.pvp && typeof normalizedPartial.extra.pvp === 'object' ? normalizedPartial.extra.pvp.reinforcements.length : 0, 2)
	assert.deepEqual(normalizedPartial.extra?.pvp && typeof normalizedPartial.extra.pvp === 'object' ? normalizedPartial.extra.pvp.wall : undefined, { level: 10 })
	assert.deepEqual(normalizedPartial.rewards, {
		resources: { wood: 100 }, drops: [{ type: 'item', itemId: 'token', name: '征战令', amount: 2 }], cityGold: 5,
		generalExp: 30, generalLevelBefore: 1, generalLevelAfter: 2, overflow: { wood: 3 },
	})

	const standardTrait = { traitId: 'standard_only', traitName: '标准时间线', ownerSide: 'primary', ownerRole: 'attacker', detail: { extraDamage: 7 } }
	const normalizedStandard = normalizeBattleReportDetail({
		...report,
		detail: { ...detail, traits: [standardTrait], rewards: { resources: { wood: 7 }, drops: [], cityGold: 0, generalExp: 0, generalLevelBefore: 0, generalLevelAfter: 0, overflow: {} } },
	})
	assert.deepEqual(normalizedStandard.traits, [standardTrait])
	assert.deepEqual(normalizedStandard.rewards, { resources: { wood: 7 }, drops: [], cityGold: 0, generalExp: 0, generalLevelBefore: 0, generalLevelAfter: 0, overflow: {} })
})

test('历史防守与援军战报恢复正确参战角色和将领快照', () => {
	const common = {
		id: 'legacy-role-report', playerId: 'defender', ownerPlayerId: 'defender', sourceType: 'player_city', battleType: 'attack',
		playerFaction: 'shu', playerName: '守城方', targetId: 'attacker', targetName: '来袭方', defenderFaction: 'wei',
		result: 'attacker_victory', rewards: {}, read: true, createdAt: '2026-07-19T12:00:00Z',
	}
	const defense = buildLegacyBattleReportDetail({
		...common, viewType: 'defense', ownerSide: 'defender', type: 'defense', playerPower: 1000, enemyPower: 2000,
		dispatchedUnits: { greedyWolf: 100 }, lostUnits: { greedyWolf: 20 }, survivedUnits: { greedyWolf: 80 },
		defenderUnits: { huWei: 200 }, defenderLostUnits: { huWei: 50 }, defenderRevealed: true,
		pvpAttackerGenerals: [{ id: 'caocao', name: '曹操', level: 1 }],
		pvpDefenderGenerals: [{ id: 'liubei', name: '刘备', level: 1 }],
		pvpWall: { faction: 'shu', level: 10, base: 1, multiplier: 1.2, factionDefenseBonus: 0.02, totalDefenseBonus: 0.2, hardness: 1.35, minDamagedLevelFrom20: 16, maxDamagedLevelFrom20: 17 },
	})
	assert.equal(defense.primarySide.role, 'attacker')
	assert.equal(defense.primarySide.cityName, '来袭方')
	assert.equal(defense.primarySide.generals?.[0]?.id, 'caocao')
	assert.deepEqual(defense.primarySide.units[0], { unitType: 'huWei', amountBefore: 200, dispatched: 200, lost: 50, survived: 150 })
	assert.equal(defense.secondarySide?.role, 'defender')
	assert.equal(defense.secondarySide?.cityName, '守城方')
	assert.equal(defense.secondarySide?.generals?.[0]?.id, 'liubei')
	assert.deepEqual(defense.secondarySide?.units[0], { unitType: 'greedyWolf', amountBefore: 100, dispatched: 100, lost: 20, survived: 80 })
	assert.deepEqual(defense.extra?.pvp && typeof defense.extra.pvp === 'object' ? defense.extra.pvp.wall : undefined, {
		faction: 'shu', level: 10, base: 1, multiplier: 1.2, factionDefenseBonus: 0.02, totalDefenseBonus: 0.2, hardness: 1.35, minDamagedLevelFrom20: 16, maxDamagedLevelFrom20: 17,
	})

	const reinforcement = buildLegacyBattleReportDetail({
		...common, playerId: 'helper', ownerPlayerId: 'helper', playerName: '援军方', viewType: 'reinforcement', ownerSide: 'reinforcement',
		type: 'reinforce', dispatchedUnits: { greedyWolf: 100 }, lostUnits: { greedyWolf: 80 }, survivedUnits: { greedyWolf: 20 }, defenderRevealed: true,
		pvpReinforcements: [{ reinforcementId: 'rein_legacy', fromPlayerId: 'helper', fromPlayerName: '援军方', faction: 'shu', troops: { greedyWolf: 100 }, generals: [{ id: 'zhaoyun', name: '赵云', level: 1 }] }],
		pvpReinforcementLosses: { rein_legacy: { greedyWolf: 80 } },
	})
	assert.equal(reinforcement.primarySide.role, 'reinforcement')
	assert.equal(reinforcement.primarySide.generals?.[0]?.id, 'zhaoyun')
	assert.equal(reinforcement.primarySide.units[0].survived, 20)
	assert.equal(reinforcement.secondarySide, undefined)
})

test('锦帆奇袭明确限制为掠夺战战斗前攻击加成', () => {
	const trait = getTraitMeta('jinfan_qixi')
	assert.equal(trait.trigger, '掠夺战战斗前')
	assert.match(trait.description, /仅在掠夺战/)
	assert.match(trait.description, /10%/)
	assert.match(trait.description, /普通进攻无效/)
	assert.equal(formatTraitOutcomeDetail('attackModifiedUnits', { overlordRider: 1 }), '实际攻击修正: 霸王骑 +1')
})

test('甘宁掠夺战双特性同时展示真实兵损、攻击修正和最终资源增量', () => {
	const options = {
		faction: 'wu',
		units: {
			wu: { wuInfantry: { name: '吴步兵' } },
			wei: { weiInfantry: { name: '魏步兵' } },
		},
	}
	const detail = {
		primarySide: {
			power: 1100,
			units: [{ unitType: 'wuInfantry', unitName: '吴步兵', amountBefore: 100, dispatched: 100, lost: 48, survived: 52 }],
		},
		secondarySide: {
			power: 1050,
			units: [{ unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 105, dispatched: 105, lost: 54, survived: 51 }],
		},
		rewards: { generalExp: 54, resources: { wood: 312 } },
		traits: [
			{
				traitId: 'jinfan_qixi', traitName: '锦帆奇袭', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'ganning',
				detail: { attackBonusRate: 0.1, attackModifiedUnits: { wuInfantry: 1 }, triggerChance: 1 },
			},
			{
				traitId: 'jinfan_jielue', traitName: '锦帆劫掠', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'ganning',
				detail: { plunderBonusRate: 0.2, plunderDelta: { wood: 52 }, triggerChance: 1 },
			},
		],
	}
	assert.equal(hasStandardUnitRows(detail.primarySide), true)
	assert.equal(hasStandardUnitRows(detail.secondarySide), true)
	assert.deepEqual(detail.primarySide.units[0], { unitType: 'wuInfantry', unitName: '吴步兵', amountBefore: 100, dispatched: 100, lost: 48, survived: 52 })
	assert.deepEqual(detail.secondarySide.units[0], { unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 105, dispatched: 105, lost: 54, survived: 51 })
	assert.deepEqual(detail.traits.map((trait) => trait.traitId), ['jinfan_qixi', 'jinfan_jielue'])
	assert.deepEqual(detail.traits.map((trait) => getTraitMeta(trait.traitId).trigger), ['掠夺战战斗前', '掠夺结算时'])
	assert.deepEqual(detail.traits.map((trait) => resolveReportTraitDisplaySide({ sourceType: 'player_city', viewType: 'attack' }, trait)), ['primary', 'primary'])
	assert.equal(formatTraitOutcomeDetail('attackBonusRate', detail.traits[0].detail.attackBonusRate), '设计攻击加成: 10%')
	assert.equal(formatTraitOutcomeDetail('attackModifiedUnits', detail.traits[0].detail.attackModifiedUnits, options), '实际攻击修正: 吴步兵 +1')
	assert.equal(formatTraitOutcomeDetail('plunderBonusRate', detail.traits[1].detail.plunderBonusRate), '设计掠夺修正: 20%')
	assert.equal(formatTraitOutcomeDetail('plunderDelta', detail.traits[1].detail.plunderDelta), '掠夺资源修正: 木材 +52')
	assert.deepEqual(detail.rewards, { generalExp: 54, resources: { wood: 312 } })
})

test('孙权掠夺防守双特性同时归属防守侧并展示真实防御和资源减量', () => {
	const options = {
		faction: 'wu',
		units: {
			wei: { weiInfantry: { name: '魏步兵' } },
			wu: { wuInfantry: { name: '吴步兵' } },
		},
	}
	const detail = {
		primarySide: {
			power: 2000,
			units: [{ unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 200, dispatched: 200, lost: 79, survived: 121 }],
		},
		secondarySide: {
			power: 1500,
			units: [{ unitType: 'wuInfantry', unitName: '吴步兵', amountBefore: 100, dispatched: 100, lost: 60, survived: 40 }],
		},
		rewards: { generalExp: 60, resources: { wood: 484 } },
		traits: [
			{
				traitId: 'jiangdong_gushou', traitName: '江东固守', ownerSide: 'secondary', ownerRole: 'defender', generalId: 'sunquan',
				detail: { defenseBonusRate: 0.5, infantryDefenseModifiedUnits: { wuInfantry: 5 }, cavalryDefenseModifiedUnits: { wuInfantry: 4 }, triggerChance: 1 },
			},
			{
				traitId: 'jiangdong_haoling', traitName: '江东号令', ownerSide: 'secondary', ownerRole: 'defender', generalId: 'sunquan',
				detail: { plunderBonusRate: -0.2, plunderDelta: { wood: -121 } },
			},
		],
	}
	assert.equal(hasStandardUnitRows(detail.primarySide), true)
	assert.equal(hasStandardUnitRows(detail.secondarySide), true)
	assert.deepEqual(detail.primarySide.units[0], { unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 200, dispatched: 200, lost: 79, survived: 121 })
	assert.deepEqual(detail.secondarySide.units[0], { unitType: 'wuInfantry', unitName: '吴步兵', amountBefore: 100, dispatched: 100, lost: 60, survived: 40 })
	assert.deepEqual(detail.traits.map((trait) => trait.traitId), ['jiangdong_gushou', 'jiangdong_haoling'])
	assert.deepEqual(detail.traits.map((trait) => getTraitMeta(trait.traitId).trigger), ['防守/增援战斗前', '掠夺结算时'])
	assert.deepEqual(detail.traits.map((trait) => resolveReportTraitDisplaySide({ sourceType: 'player_city', viewType: 'attack' }, trait)), ['secondary', 'secondary'])
	assert.equal(formatTraitOutcomeDetail('defenseBonusRate', detail.traits[0].detail.defenseBonusRate), '设计防御加成: 50%')
	assert.equal(formatTraitOutcomeDetail('infantryDefenseModifiedUnits', detail.traits[0].detail.infantryDefenseModifiedUnits, options), '实际步防修正: 吴步兵 +5')
	assert.equal(formatTraitOutcomeDetail('cavalryDefenseModifiedUnits', detail.traits[0].detail.cavalryDefenseModifiedUnits, options), '实际骑防修正: 吴步兵 +4')
	assert.equal(formatTraitOutcomeDetail('plunderBonusRate', detail.traits[1].detail.plunderBonusRate), '设计掠夺修正: -20%')
	assert.equal(formatTraitOutcomeDetail('plunderDelta', detail.traits[1].detail.plunderDelta), '掠夺资源修正: 木材 -121')
	assert.deepEqual(detail.rewards, { generalExp: 60, resources: { wood: 484 } })
})

test('孙权进攻和甘宁防守时拥有快照不会补造掠夺特性', () => {
	const detail = {
		primarySide: {
			role: 'attacker', power: 10000,
			generals: [{
				id: 'sunquan', name: '孙权', level: 1,
				traits: [{ traitId: 'jiangdong_haoling', name: '江东号令' }],
			}],
			units: [{ unitType: 'wuInfantry', unitName: '吴步兵', amountBefore: 1000, dispatched: 1000, lost: 1, survived: 999 }],
		},
		secondarySide: {
			role: 'defender', power: 100,
			generals: [{
				id: 'ganning', name: '甘宁', level: 1,
				traits: [{ traitId: 'jinfan_jielue', name: '锦帆劫掠' }, { traitId: 'jinfan_qixi', name: '锦帆奇袭' }],
			}],
			units: [{ unitType: 'wuInfantry', unitName: '吴步兵', amountBefore: 10, dispatched: 10, lost: 9, survived: 1 }],
		},
		rewards: { generalExp: 9, resources: { wood: 4995 } },
		traits: [],
	}
	assert.equal(hasStandardUnitRows(detail.primarySide), true)
	assert.equal(hasStandardUnitRows(detail.secondarySide), true)
	assert.equal(hasTraitEntries(detail), false)
	assert.deepEqual(detail.primarySide.generals[0].traits.map((trait) => trait.traitId), ['jiangdong_haoling'])
	assert.deepEqual(detail.secondarySide.generals[0].traits.map((trait) => trait.traitId), ['jinfan_jielue', 'jinfan_qixi'])
	assert.deepEqual(detail.primarySide.units[0], { unitType: 'wuInfantry', unitName: '吴步兵', amountBefore: 1000, dispatched: 1000, lost: 1, survived: 999 })
	assert.deepEqual(detail.secondarySide.units[0], { unitType: 'wuInfantry', unitName: '吴步兵', amountBefore: 10, dispatched: 10, lost: 9, survived: 1 })
	assert.deepEqual(detail.rewards, { generalExp: 9, resources: { wood: 4995 } })
})

test('黄盖同阶段双特性只移除敌方后续特性并保留自身真实反击', () => {
	const options = {
		faction: 'wu',
		units: {
			wu: { wuInfantry: { name: '吴步兵' } },
			shu: { shuInfantry: { name: '蜀步兵' } },
		},
	}
	const detail = {
		primarySide: {
			power: 10000,
			units: [{ unitType: 'wuInfantry', unitName: '吴步兵', amountBefore: 1000, dispatched: 1000, lost: 500, survived: 500 }],
		},
		secondarySide: {
			power: 10000,
			units: [{ unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 1000, dispatched: 1000, lost: 600, survived: 400 }],
		},
		rewards: { generalExp: 600, resources: {} },
		traits: [
			{
				traitId: 'kurouji', traitName: '苦肉计', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'huanggai',
				detail: { disableTraitCount: 1, disabledTraitCount: 1, triggerChance: 1 },
			},
			{
				traitId: 'kurou_fanji', traitName: '苦肉反击', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'huanggai',
				detail: { effectRate: 0.1, extraLosses: { shuInfantry: 100 } },
			},
		],
	}
	assert.equal(hasStandardUnitRows(detail.primarySide), true)
	assert.equal(hasStandardUnitRows(detail.secondarySide), true)
	assert.deepEqual(detail.primarySide.units[0], { unitType: 'wuInfantry', unitName: '吴步兵', amountBefore: 1000, dispatched: 1000, lost: 500, survived: 500 })
	assert.deepEqual(detail.secondarySide.units[0], { unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 1000, dispatched: 1000, lost: 600, survived: 400 })
	assert.deepEqual(detail.traits.map((trait) => trait.traitId), ['kurouji', 'kurou_fanji'])
	assert.equal(detail.traits.some((trait) => trait.traitId === 'laodang_yizhuang'), false)
	assert.deepEqual(detail.traits.map((trait) => getTraitMeta(trait.traitId).trigger), ['战斗结算后', '战斗结算后'])
	assert.deepEqual(detail.traits.map((trait) => resolveReportTraitDisplaySide({ sourceType: 'player_city', viewType: 'attack' }, trait)), ['primary', 'primary'])
	assert.equal(formatTraitOutcomeDetail('disableTraitCount', detail.traits[0].detail.disableTraitCount), '设计压制特性数: 1')
	assert.equal(formatTraitOutcomeDetail('disabledTraitCount', detail.traits[0].detail.disabledTraitCount), '实际压制特性数: 1')
	assert.equal(formatTraitOutcomeDetail('effectRate', detail.traits[1].detail.effectRate), '设计效果比例: 10%')
	assert.equal(formatTraitOutcomeDetail('extraLosses', detail.traits[1].detail.extraLosses, options), '追加损失: 蜀步兵 +100')
	assert.deepEqual(detail.rewards, { generalExp: 600, resources: {} })
})

test('黄盖苦肉计未命中时双方后续伤害都正常生效', () => {
	const detail = {
		sourceType: 'player_city', viewType: 'attack', battleType: 'plunder',
		primarySide: {
			role: 'attacker', power: 10000,
			generals: [{
				id: 'huanggai', name: '黄盖', level: 1,
				traits: [{ traitId: 'kurouji', name: '苦肉计' }, { traitId: 'kurou_fanji', name: '苦肉反击' }],
			}],
			units: [{ unitType: 'wuInfantry', unitName: '吴步兵', amountBefore: 1000, dispatched: 1000, lost: 600, survived: 400 }],
		},
		secondarySide: {
			role: 'defender', power: 10000,
			generals: [{ id: 'huangzhong', name: '黄忠', level: 1, traits: [{ traitId: 'laodang_yizhuang', name: '老当益壮' }] }],
			units: [{ unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 1000, dispatched: 1000, lost: 600, survived: 400 }],
		},
		rewards: { generalExp: 600, resources: {} },
		traits: [
			{
				traitId: 'kurou_fanji', traitName: '苦肉反击', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'huanggai',
				detail: { effectRate: 0.1, extraLosses: { shuInfantry: 100 } },
			},
			{
				traitId: 'laodang_yizhuang', traitName: '老当益壮', ownerSide: 'secondary', ownerRole: 'defender', generalId: 'huangzhong',
				detail: { effectRate: 0.1, extraLosses: { wuInfantry: 100 } },
			},
		],
	}
	const options = { faction: 'wu', units: { wu: { wuInfantry: { name: '吴步兵' } }, shu: { shuInfantry: { name: '蜀步兵' } } } }
	assert.deepEqual(detail.primarySide.generals[0].traits.map((trait) => trait.traitId), ['kurouji', 'kurou_fanji'])
	assert.deepEqual(detail.traits.map((trait) => trait.traitId), ['kurou_fanji', 'laodang_yizhuang'])
	assert.equal(detail.traits.some((trait) => trait.traitId === 'kurouji'), false)
	assert.deepEqual(detail.traits.map((trait) => resolveReportTraitDisplaySide(detail, trait)), ['primary', 'secondary'])
	assert.deepEqual(detail.primarySide.units[0], { unitType: 'wuInfantry', unitName: '吴步兵', amountBefore: 1000, dispatched: 1000, lost: 600, survived: 400 })
	assert.deepEqual(detail.secondarySide.units[0], { unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 1000, dispatched: 1000, lost: 600, survived: 400 })
	assert.equal(formatTraitOutcomeDetail('extraLosses', detail.traits[0].detail.extraLosses, options), '追加损失: 蜀步兵 +100')
	assert.equal(formatTraitOutcomeDetail('extraLosses', detail.traits[1].detail.extraLosses, options), '追加损失: 吴步兵 +100')
})

test('防守黄盖苦肉计未命中时按攻守顺序保留双方后续伤害', () => {
	const detail = {
		sourceType: 'player_city', viewType: 'defense', battleType: 'plunder', ownerSide: 'defender',
		primarySide: {
			role: 'attacker', power: 10000,
			generals: [{ id: 'huangzhong', name: '黄忠', level: 1, traits: [{ traitId: 'laodang_yizhuang', name: '老当益壮' }] }],
			units: [{ unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 1000, dispatched: 1000, lost: 600, survived: 400 }],
		},
		secondarySide: {
			role: 'defender', power: 10000,
			generals: [{
				id: 'huanggai', name: '黄盖', level: 1,
				traits: [{ traitId: 'kurouji', name: '苦肉计' }, { traitId: 'kurou_fanji', name: '苦肉反击' }],
			}],
			units: [{ unitType: 'wuInfantry', unitName: '吴步兵', amountBefore: 1000, dispatched: 1000, lost: 600, survived: 400 }],
		},
		rewards: { generalExp: 600, resources: {} },
		traits: [
			{
				traitId: 'laodang_yizhuang', traitName: '老当益壮', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'huangzhong',
				detail: { effectRate: 0.1, extraLosses: { wuInfantry: 100 } },
			},
			{
				traitId: 'kurou_fanji', traitName: '苦肉反击', ownerSide: 'secondary', ownerRole: 'defender', generalId: 'huanggai',
				detail: { effectRate: 0.1, extraLosses: { shuInfantry: 100 } },
			},
		],
	}
	const options = { faction: 'wu', units: { wu: { wuInfantry: { name: '吴步兵' } }, shu: { shuInfantry: { name: '蜀步兵' } } } }
	assert.deepEqual(detail.secondarySide.generals[0].traits.map((trait) => trait.traitId), ['kurouji', 'kurou_fanji'])
	assert.deepEqual(detail.traits.map((trait) => trait.traitId), ['laodang_yizhuang', 'kurou_fanji'])
	assert.equal(detail.traits.some((trait) => trait.traitId === 'kurouji'), false)
	assert.deepEqual(detail.traits.map((trait) => resolveReportTraitDisplaySide(detail, trait)), ['primary', 'secondary'])
	assert.deepEqual(detail.primarySide.units[0], { unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 1000, dispatched: 1000, lost: 600, survived: 400 })
	assert.deepEqual(detail.secondarySide.units[0], { unitType: 'wuInfantry', unitName: '吴步兵', amountBefore: 1000, dispatched: 1000, lost: 600, survived: 400 })
	assert.equal(formatTraitOutcomeDetail('extraLosses', detail.traits[0].detail.extraLosses, options), '追加损失: 吴步兵 +100')
	assert.equal(formatTraitOutcomeDetail('extraLosses', detail.traits[1].detail.extraLosses, options), '追加损失: 蜀步兵 +100')
	assert.deepEqual(detail.rewards, { generalExp: 600, resources: {} })
})

test('双方苦肉计同时触发时保留两条同名压制并按攻守分侧', () => {
	const detail = {
		primarySide: {
			role: 'attacker', power: 1000,
			generals: [{ id: 'attacker_general', name: '进攻将领', level: 1 }],
			units: [{ unitType: 'wuInfantry', unitName: '吴步兵', amountBefore: 100, dispatched: 100, lost: 50, survived: 50 }],
		},
		secondarySide: {
			role: 'defender', power: 1020,
			generals: [{ id: 'defender_general', name: '防守将领', level: 1 }],
			units: [{ unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 100, dispatched: 100, lost: 49, survived: 51 }],
		},
		rewards: { generalExp: 49, resources: {} },
		traits: [
			{
				traitId: 'kurouji', traitName: '苦肉计', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'attacker_general',
				detail: { disableTraitCount: 1, disabledTraitCount: 1, triggerChance: 1 },
			},
			{
				traitId: 'kurouji', traitName: '苦肉计', ownerSide: 'secondary', ownerRole: 'defender', generalId: 'defender_general',
				detail: { disableTraitCount: 1, disabledTraitCount: 1, triggerChance: 1 },
			},
		],
	}
	assert.deepEqual(detail.traits.map((trait) => trait.traitId), ['kurouji', 'kurouji'])
	assert.deepEqual(detail.traits.map((trait) => resolveReportTraitDisplaySide({ sourceType: 'player_city', viewType: 'attack' }, trait)), ['primary', 'secondary'])
	assert.deepEqual(detail.traits.map((trait) => getTraitMeta(trait.traitId).trigger), ['战斗结算后', '战斗结算后'])
	assert.deepEqual(detail.traits.map((trait) => formatTraitOutcomeDetail('disabledTraitCount', trait.detail.disabledTraitCount)), ['实际压制特性数: 1', '实际压制特性数: 1'])
	assert.equal(detail.traits.some((trait) => trait.traitId === 'kurou_fanji' || trait.traitId === 'laodang_yizhuang'), false)
	assert.deepEqual(detail.primarySide.units[0], { unitType: 'wuInfantry', unitName: '吴步兵', amountBefore: 100, dispatched: 100, lost: 50, survived: 50 })
	assert.deepEqual(detail.secondarySide.units[0], { unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 100, dispatched: 100, lost: 49, survived: 51 })
})

test('马超被动武力只进入参战快照和战力，触发时间线仅展示西凉突击', () => {
	const options = {
		faction: 'shu',
		units: {
			shu: { shuInfantry: { name: '蜀步兵' } },
			wei: { weiCavalry: { name: '魏骑兵' } },
		},
	}
	const detail = {
		primarySide: {
			power: 14000,
			generals: [{
				id: 'machao', name: '马超', level: 1,
				traits: [{ traitId: 'xiliang_tuji', name: '西凉突击' }, { traitId: 'tianshen_xiafan', name: '天神下凡' }],
			}],
			units: [{ unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 1000, dispatched: 1000, lost: 382, survived: 618 }],
		},
		secondarySide: {
			power: 10000,
			units: [{ unitType: 'weiCavalry', unitName: '魏骑兵', amountBefore: 1000, dispatched: 1000, lost: 737, survived: 263 }],
		},
		rewards: { generalExp: 737, generalLevelBefore: 1, generalLevelAfter: 2, resources: {} },
		traits: [{
			traitId: 'xiliang_tuji', traitName: '西凉突击', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'machao',
			detail: { effectRate: 0.12, targetExtraLosses: { weiCavalry: 120 }, triggerChance: 1 },
		}],
	}
	assert.equal(hasStandardUnitRows(detail.primarySide), true)
	assert.equal(hasStandardUnitRows(detail.secondarySide), true)
	assert.deepEqual(detail.primarySide.generals[0].traits.map((trait) => trait.traitId), ['xiliang_tuji', 'tianshen_xiafan'])
	assert.deepEqual(detail.traits.map((trait) => trait.traitId), ['xiliang_tuji'])
	assert.equal(getTraitMeta('tianshen_xiafan').trigger, '被动属性')
	assert.equal(getTraitMeta('xiliang_tuji').trigger, '战斗结算后')
	assert.equal(resolveReportTraitDisplaySide({ sourceType: 'player_city', viewType: 'attack' }, detail.traits[0]), 'primary')
	assert.equal(formatTraitOutcomeDetail('effectRate', detail.traits[0].detail.effectRate), '设计效果比例: 12%')
	assert.equal(formatTraitOutcomeDetail('targetExtraLosses', detail.traits[0].detail.targetExtraLosses, options), '目标兵种追加损失: 魏骑兵 +120')
	assert.deepEqual(detail.rewards, { generalExp: 737, generalLevelBefore: 1, generalLevelAfter: 2, resources: {} })
})

test('天神下凡生效但西凉突击未命中时保留被动战力且触发区为空', () => {
	const detail = {
		primarySide: {
			power: 14000,
			generals: [{
				id: 'machao', name: '马超', level: 1, stats: { force: 0 }, effectiveStats: { force: 20 }, buffs: { attackBonus: 0.4 },
				traits: [{ traitId: 'xiliang_tuji', name: '西凉突击' }, { traitId: 'tianshen_xiafan', name: '天神下凡' }],
			}],
			units: [{ unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 1000, dispatched: 1000, lost: 382, survived: 618 }],
		},
		secondarySide: {
			power: 10000,
			units: [{ unitType: 'weiCavalry', unitName: '魏骑兵', amountBefore: 1000, dispatched: 1000, lost: 617, survived: 383 }],
		},
		rewards: { generalExp: 617, resources: {} },
		traits: [],
	}
	assert.deepEqual(detail.primarySide.generals[0].traits.map((trait) => trait.traitId), ['xiliang_tuji', 'tianshen_xiafan'])
	assert.equal(detail.primarySide.generals[0].effectiveStats.force - detail.primarySide.generals[0].stats.force, 20)
	assert.equal(detail.primarySide.generals[0].buffs.attackBonus, 0.4)
	assert.equal(detail.primarySide.power, 14000)
	assert.equal(hasTraitEntries(detail), false)
	assert.deepEqual(detail.primarySide.units[0], { unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 1000, dispatched: 1000, lost: 382, survived: 618 })
	assert.deepEqual(detail.secondarySide.units[0], { unitType: 'weiCavalry', unitName: '魏骑兵', amountBefore: 1000, dispatched: 1000, lost: 617, survived: 383 })
	assert.deepEqual(detail.rewards, { generalExp: 617, resources: {} })
})

test('魏延进攻破防与防守加防按方向互斥并对齐完整战报数值', () => {
	const options = {
		faction: 'shu',
		units: {
			shu: { shuInfantry: { name: '蜀步兵' } },
			wei: { weiInfantry: { name: '魏步兵' } },
		},
	}
	const ownedTraits = [{ traitId: 'qibing_raohou', name: '奇兵绕后' }, { traitId: 'gushou_hanzhong', name: '固守汉中' }]
	const attackDetail = {
		primarySide: {
			power: 10000,
			generals: [{ id: 'weiyan', name: '魏延', level: 1, traits: ownedTraits }],
			units: [{ unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 1000, dispatched: 1000, lost: 421, survived: 579 }],
		},
		secondarySide: {
			power: 8000,
			units: [{ unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 1000, dispatched: 1000, lost: 578, survived: 422 }],
		},
		rewards: { generalExp: 578, generalLevelBefore: 1, generalLevelAfter: 2, resources: {} },
		traits: [{
			traitId: 'qibing_raohou', traitName: '奇兵绕后', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'weiyan',
			detail: { enemyDefenseReductionRate: 0.2, infantryDefenseModifiedUnits: { weiInfantry: -2 }, cavalryDefenseModifiedUnits: { weiInfantry: -2 }, triggerChance: 1 },
		}],
	}
	assert.equal(hasStandardUnitRows(attackDetail.primarySide), true)
	assert.equal(hasStandardUnitRows(attackDetail.secondarySide), true)
	assert.deepEqual(attackDetail.primarySide.generals[0].traits.map((trait) => trait.traitId), ['qibing_raohou', 'gushou_hanzhong'])
	assert.deepEqual(attackDetail.traits.map((trait) => trait.traitId), ['qibing_raohou'])
	assert.equal(attackDetail.traits.some((trait) => trait.traitId === 'gushou_hanzhong'), false)
	assert.deepEqual(attackDetail.primarySide.units[0], { unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 1000, dispatched: 1000, lost: 421, survived: 579 })
	assert.deepEqual(attackDetail.secondarySide.units[0], { unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 1000, dispatched: 1000, lost: 578, survived: 422 })
	assert.equal(resolveReportTraitDisplaySide({ sourceType: 'player_city', viewType: 'attack' }, attackDetail.traits[0]), 'primary')
	assert.equal(formatTraitOutcomeDetail('enemyDefenseReductionRate', attackDetail.traits[0].detail.enemyDefenseReductionRate), '设计敌方防御降低: 20%')
	assert.equal(formatTraitOutcomeDetail('infantryDefenseModifiedUnits', attackDetail.traits[0].detail.infantryDefenseModifiedUnits, options), '实际步防修正: 魏步兵 -2')
	assert.equal(formatTraitOutcomeDetail('cavalryDefenseModifiedUnits', attackDetail.traits[0].detail.cavalryDefenseModifiedUnits, options), '实际骑防修正: 魏步兵 -2')
	assert.deepEqual(attackDetail.rewards, { generalExp: 578, generalLevelBefore: 1, generalLevelAfter: 2, resources: {} })

	const defenseDetail = {
		primarySide: {
			power: 10000,
			units: [{ unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 1000, dispatched: 1000, lost: 826, survived: 174 }],
		},
		secondarySide: {
			power: 30000,
			generals: [{ id: 'weiyan', name: '魏延', level: 1, traits: ownedTraits }],
			units: [{ unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 1000, dispatched: 1000, lost: 173, survived: 827 }],
		},
		rewards: { generalExp: 826, generalLevelBefore: 1, generalLevelAfter: 2, resources: {} },
		traits: [{
			traitId: 'gushou_hanzhong', traitName: '固守汉中', ownerSide: 'secondary', ownerRole: 'defender', generalId: 'weiyan',
			detail: { generalDefenseFlat: 20, infantryDefenseModifiedUnits: { shuInfantry: 20 }, cavalryDefenseModifiedUnits: { shuInfantry: 20 }, triggerChance: 1 },
		}],
	}
	assert.equal(hasStandardUnitRows(defenseDetail.primarySide), true)
	assert.equal(hasStandardUnitRows(defenseDetail.secondarySide), true)
	assert.deepEqual(defenseDetail.secondarySide.generals[0].traits.map((trait) => trait.traitId), ['qibing_raohou', 'gushou_hanzhong'])
	assert.deepEqual(defenseDetail.traits.map((trait) => trait.traitId), ['gushou_hanzhong'])
	assert.equal(defenseDetail.traits.some((trait) => trait.traitId === 'qibing_raohou'), false)
	assert.deepEqual(defenseDetail.primarySide.units[0], { unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 1000, dispatched: 1000, lost: 826, survived: 174 })
	assert.deepEqual(defenseDetail.secondarySide.units[0], { unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 1000, dispatched: 1000, lost: 173, survived: 827 })
	assert.equal(resolveReportTraitDisplaySide({ sourceType: 'player_city', viewType: 'defense' }, defenseDetail.traits[0]), 'secondary')
	assert.equal(formatTraitOutcomeDetail('generalDefenseFlat', defenseDetail.traits[0].detail.generalDefenseFlat), '设计全军防御增加: 20')
	assert.equal(formatTraitOutcomeDetail('infantryDefenseModifiedUnits', defenseDetail.traits[0].detail.infantryDefenseModifiedUnits, options), '实际步防修正: 蜀步兵 +20')
	assert.equal(formatTraitOutcomeDetail('cavalryDefenseModifiedUnits', defenseDetail.traits[0].detail.cavalryDefenseModifiedUnits, options), '实际骑防修正: 蜀步兵 +20')
	assert.deepEqual(defenseDetail.rewards, { generalExp: 826, generalLevelBefore: 1, generalLevelAfter: 2, resources: {} })
})

test('魏延奇兵绕后合法未命中时只展示基础战斗结果', () => {
	const ownedTraits = [{ traitId: 'qibing_raohou', name: '奇兵绕后' }, { traitId: 'gushou_hanzhong', name: '固守汉中' }]
	const detail = {
		sourceType: 'player_city', viewType: 'attack', battleType: 'plunder',
		primarySide: {
			role: 'attacker', power: 10000,
			generals: [{ id: 'weiyan', name: '魏延', level: 1, traits: ownedTraits }],
			units: [{ unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 1000, dispatched: 1000, lost: 500, survived: 500 }],
		},
		secondarySide: {
			role: 'defender', power: 10000,
			generals: [{ id: 'caocao', name: '曹操', level: 1 }],
			units: [{ unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 1000, dispatched: 1000, lost: 500, survived: 500 }],
		},
		rewards: { generalExp: 500, resources: {} },
		traits: [],
	}
	assert.equal(hasStandardUnitRows(detail.primarySide), true)
	assert.equal(hasStandardUnitRows(detail.secondarySide), true)
	assert.equal(hasTraitEntries(detail), false)
	assert.deepEqual(detail.primarySide.generals[0].traits.map((trait) => trait.traitId), ['qibing_raohou', 'gushou_hanzhong'])
	assert.deepEqual(detail.primarySide.units[0], { unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 1000, dispatched: 1000, lost: 500, survived: 500 })
	assert.deepEqual(detail.secondarySide.units[0], { unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 1000, dispatched: 1000, lost: 500, survived: 500 })
	assert.deepEqual(detail.rewards, { generalExp: 500, resources: {} })
})

test('夏侯渊被动不进入触发时间线且盾阵对齐援军实损', () => {
	const options = { faction: 'wei', units: { wei: { qiQiYing: { name: '骁骑营' } } } }
	const detail = {
		primarySide: {
			role: 'reinforcement', power: 1710,
			generals: [{
				id: 'xiahouyuan', name: '夏侯渊', level: 1,
				traits: [{ traitId: 'jixing_benxi', name: '疾行奔袭', params: { unitAttackFlat: 18, unitSpeedFlat: 5 } }, { traitId: 'dunzhen_fangyu', name: '盾阵防御' }],
			}],
			units: [{ unitType: 'qiQiYing', unitName: '骁骑营', amountBefore: 100, dispatched: 100, lost: 60, survived: 40 }],
		},
		rewards: { generalExp: 110, resources: {} },
		traits: [{
			traitId: 'dunzhen_fangyu', traitName: '盾阵防御', ownerSide: 'reinforcement', ownerRole: 'reinforcement', generalId: 'xiahouyuan',
			detail: { defenseBonusRate: 0.3, infantryDefenseModifiedUnits: { qiQiYing: 4 }, cavalryDefenseModifiedUnits: { qiQiYing: 3 }, triggerChance: 0.6 },
		}],
	}
	assert.equal(hasStandardUnitRows(detail.primarySide), true)
	assert.deepEqual(detail.primarySide.units[0], { unitType: 'qiQiYing', unitName: '骁骑营', amountBefore: 100, dispatched: 100, lost: 60, survived: 40 })
	assert.deepEqual(detail.primarySide.generals[0].traits.map((trait) => trait.traitId), ['jixing_benxi', 'dunzhen_fangyu'])
	assert.deepEqual(detail.traits.map((trait) => trait.traitId), ['dunzhen_fangyu'])
	assert.equal(detail.traits.some((trait) => trait.traitId === 'jixing_benxi'), false)
	assert.equal(getTraitMeta('jixing_benxi').trigger, '永久被动')
	assert.equal(getTraitMeta('dunzhen_fangyu').trigger, '防守/增援战斗前')
	assert.equal(resolveReportTraitDisplaySide({ sourceType: 'player_city', viewType: 'reinforcement' }, detail.traits[0]), 'reinforcement')
	assert.equal(formatTraitOutcomeDetail('defenseBonusRate', detail.traits[0].detail.defenseBonusRate), '设计防御加成: 30%')
	assert.equal(formatTraitOutcomeDetail('infantryDefenseModifiedUnits', detail.traits[0].detail.infantryDefenseModifiedUnits, options), '实际步防修正: 骁骑营 +4')
	assert.equal(formatTraitOutcomeDetail('cavalryDefenseModifiedUnits', detail.traits[0].detail.cavalryDefenseModifiedUnits, options), '实际骑防修正: 骁骑营 +3')
	assert.equal(formatTraitOutcomeDetail('unitAttackFlat', 18), '设计单位攻击增加: 18')
	assert.equal(formatTraitOutcomeDetail('unitSpeedFlat', 5), '设计单位移动增加: 5')
	const detailSource = readFileSync(new URL('../src/pages/news/components/BattleReportDetail.tsx', import.meta.url), 'utf8')
	assert.match(detailSource, /jixing_benxi: \{ name: '疾行奔袭', unitName: '骁骑营' \}/)
	assert.deepEqual(detail.rewards, { generalExp: 110, resources: {} })
})

test('援军对敌特性归属援军并展示后端实际震慑、扣兵和压制数', () => {
	const options = { faction: 'wei', units: { wei: { weiInfantry: { name: '魏步兵' } } } }
	const report = { sourceType: 'player_city', viewType: 'attack' }
	const traits = [
		{ traitId: 'weizhen_zhenhe', ownerSide: 'reinforcement', ownerRole: 'reinforcement', generalId: 'zhangliao', detail: { suppressedUnits: { weiInfantry: 22 } } },
		{ traitId: 'laodang_yizhuang', ownerSide: 'reinforcement', ownerRole: 'reinforcement', generalId: 'huangzhong', detail: { extraLosses: { weiInfantry: 11 } } },
		{ traitId: 'wolong_mouzhi', ownerSide: 'reinforcement', ownerRole: 'reinforcement', generalId: 'zhugeliang', detail: { disabledTraitCount: 1 } },
	]
	for (const trait of traits) {
		assert.equal(resolveReportTraitDisplaySide(report, trait), 'reinforcement')
	}
	assert.equal(formatTraitOutcomeDetail('suppressedUnits', traits[0].detail.suppressedUnits, options), '本场压制兵力: 魏步兵 +22')
	assert.equal(formatTraitOutcomeDetail('extraLosses', traits[1].detail.extraLosses, options), '追加损失: 魏步兵 +11')
	assert.equal(formatTraitOutcomeDetail('disabledTraitCount', traits[2].detail.disabledTraitCount), '实际压制特性数: 1')
	for (const traitId of ['xiliang_tuji', 'laodang_yizhuang', 'huoshao_lianying', 'lianying_zengshang', 'kurou_fanji']) {
		assert.match(getTraitMeta(traitId).description, /作为援军/)
	}
})

test('赵云增援加速只留在拥有快照且龙胆对齐战后最终减损', () => {
	const options = { faction: 'shu', units: { shu: { shuInfantry: { name: '蜀步兵' } } } }
	const detail = {
		primarySide: {
			role: 'reinforcement', power: 1010,
			generals: [{
				id: 'zhaoyun', name: '赵云', level: 1,
				traits: [{ traitId: 'longdan_jiuyuan', name: '龙胆救援' }, { traitId: 'qijin_qichu', name: '七进七出' }],
			}],
			units: [{ unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 100, dispatched: 100, lost: 80, survived: 20 }],
		},
		rewards: { generalExp: 97, resources: {} },
		traits: [{
			traitId: 'longdan_jiuyuan', traitName: '龙胆救援', ownerSide: 'reinforcement', ownerRole: 'reinforcement', generalId: 'zhaoyun',
			detail: { lossReductionRate: 0.2, reducedLosses: { shuInfantry: 20 }, triggerChance: 1 },
		}],
	}
	assert.equal(hasStandardUnitRows(detail.primarySide), true)
	assert.deepEqual(detail.primarySide.units[0], { unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 100, dispatched: 100, lost: 80, survived: 20 })
	assert.deepEqual(detail.primarySide.generals[0].traits.map((trait) => trait.traitId), ['longdan_jiuyuan', 'qijin_qichu'])
	assert.deepEqual(detail.traits.map((trait) => trait.traitId), ['longdan_jiuyuan'])
	assert.equal(detail.traits.some((trait) => trait.traitId === 'qijin_qichu'), false)
	assert.equal(getTraitMeta('qijin_qichu').trigger, '行军创建时')
	assert.equal(getTraitMeta('longdan_jiuyuan').trigger, '防守/增援战斗结算后')
	assert.equal(resolveReportTraitDisplaySide({ sourceType: 'player_city', viewType: 'reinforcement' }, detail.traits[0]), 'reinforcement')
	assert.equal(formatTraitOutcomeDetail('lossReductionRate', detail.traits[0].detail.lossReductionRate), '设计减损比例: 20%')
	assert.equal(formatTraitOutcomeDetail('reducedLosses', detail.traits[0].detail.reducedLosses, options), '减少损失: 蜀步兵 +20')
	assert.deepEqual(detail.rewards, { generalExp: 97, resources: {} })
})

test('赵云龙胆合法未命中时保留加速快照但援军承担完整损失', () => {
	const detail = {
		primarySide: {
			role: 'reinforcement', power: 1010,
			generals: [{
				id: 'zhaoyun', name: '赵云', level: 1,
				traits: [{ traitId: 'longdan_jiuyuan', name: '龙胆救援' }, { traitId: 'qijin_qichu', name: '七进七出' }],
			}],
			units: [{ unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 100, dispatched: 100, lost: 100, survived: 0 }],
		},
		rewards: { generalExp: 97, resources: {} },
		traits: [],
	}
	assert.equal(hasStandardUnitRows(detail.primarySide), true)
	assert.equal(hasTraitEntries(detail), false)
	assert.deepEqual(detail.primarySide.generals[0].traits.map((trait) => trait.traitId), ['longdan_jiuyuan', 'qijin_qichu'])
	assert.deepEqual(detail.primarySide.units[0], { unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 100, dispatched: 100, lost: 100, survived: 0 })
	assert.deepEqual(detail.rewards, { generalExp: 97, resources: {} })
})

test('郭嘉永久四维只留在被动区且鬼才遗策展示真实阵亡和复活', () => {
	const options = { faction: 'wei', units: { wei: { weiInfantry: { name: '魏步兵' } } } }
	const detail = {
		primarySide: {
			role: 'attacker', power: 1000,
			generals: [{
				id: 'guojia', name: '郭嘉', level: 1,
				stats: { intelligence: 0, politics: 0 }, effectiveStats: { intelligence: 10, politics: 10 },
				traits: [{ traitId: 'shengui_zhicai', name: '神鬼之才', params: { politicsBonus: 10, intelligenceBonus: 10 } }, { traitId: 'guicai_yice', name: '鬼才遗策' }],
			}],
			units: [{ unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 100, dispatched: 100, lost: 100, survived: 22 }],
		},
		secondarySide: {
			role: 'defender', power: 10000,
			generals: [{ id: 'liubei', name: '刘备', level: 1 }],
			units: [{ unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 1000, dispatched: 1000, lost: 37, survived: 963 }],
		},
		rewards: { generalExp: 37, resources: {} },
		traits: [{
			traitId: 'guicai_yice', traitName: '鬼才遗策', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'guojia',
			detail: { effectRate: 0.22, actualLostUnits: { weiInfantry: 100 }, revivedUnits: { weiInfantry: 22 }, totalRevived: 22, triggerChance: 1 },
		}],
	}
	assert.equal(hasStandardUnitRows(detail.primarySide), true)
	assert.deepEqual(detail.primarySide.units[0], { unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 100, dispatched: 100, lost: 100, survived: 22 })
	assert.deepEqual(detail.primarySide.generals[0].traits.map((trait) => trait.traitId), ['shengui_zhicai', 'guicai_yice'])
	assert.deepEqual(detail.traits.map((trait) => trait.traitId), ['guicai_yice'])
	assert.equal(detail.traits.some((trait) => trait.traitId === 'shengui_zhicai'), false)
	assert.equal(getTraitMeta('shengui_zhicai').trigger, '永久被动')
	assert.equal(getTraitMeta('guicai_yice').trigger, '进攻/防守/增援战斗结束后')
	assert.equal(resolveReportTraitDisplaySide({ sourceType: 'player_city', viewType: 'attack' }, detail.traits[0]), 'primary')
	assert.equal(formatTraitOutcomeDetail('effectRate', detail.traits[0].detail.effectRate), '设计效果比例: 22%')
	assert.equal(formatTraitOutcomeDetail('actualLostUnits', detail.traits[0].detail.actualLostUnits, options), '本场真实阵亡: 魏步兵 +100')
	assert.equal(formatTraitOutcomeDetail('revivedUnits', detail.traits[0].detail.revivedUnits, options), '复活兵力: 魏步兵 +22')
	assert.equal(formatTraitOutcomeDetail('totalRevived', detail.traits[0].detail.totalRevived), '复活总数: 22')
	assert.deepEqual(detail.rewards, { generalExp: 37, resources: {} })
})

test('荀彧双留城能力只保留拥有快照且不伪造战斗触发', () => {
	const detail = {
		primarySide: {
			role: 'attacker', power: 1000,
			generals: [{
				id: 'xunyu', name: '荀彧', level: 1,
				traits: [{ traitId: 'wangzuo_zhicai', name: '王佐之才' }, { traitId: 'neizheng_jingying', name: '内政精营' }],
			}],
			units: [{ unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 100, dispatched: 100, lost: 37, survived: 63 }],
		},
		secondarySide: {
			role: 'defender', power: 500,
			generals: [{ id: 'liubei', name: '刘备', level: 1 }],
			units: [{ unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 50, dispatched: 50, lost: 50, survived: 0 }],
		},
		rewards: { generalExp: 50, resources: {} },
		traits: [],
	}
	assert.equal(hasStandardUnitRows(detail.primarySide), true)
	assert.equal(hasStandardUnitRows(detail.secondarySide), true)
	assert.deepEqual(detail.primarySide.generals[0].traits.map((trait) => trait.traitId), ['wangzuo_zhicai', 'neizheng_jingying'])
	assert.deepEqual(detail.primarySide.units[0], { unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 100, dispatched: 100, lost: 37, survived: 63 })
	assert.deepEqual(detail.secondarySide.units[0], { unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 50, dispatched: 50, lost: 50, survived: 0 })
	assert.deepEqual(detail.traits, [])
	assert.equal(getTraitMeta('wangzuo_zhicai').trigger, '留城征兵消耗时')
	assert.equal(getTraitMeta('neizheng_jingying').trigger, '留城被动生效')
	assert.deepEqual(detail.rewards, { generalExp: 50, resources: {} })
})

test('曹操产出虎卫主动进攻时战报不触发魏武统御', () => {
	const detail = {
		primarySide: {
			role: 'attacker', power: 1000,
			generals: [{
				id: 'caocao', name: '曹操', level: 1,
				traits: [{ traitId: 'weiwu_haoling', name: '魏武号令' }, { traitId: 'weiwu_tongyu', name: '魏武统御' }],
			}],
			units: [{ unitType: 'huWei', unitName: '虎卫', amountBefore: 100, dispatched: 100, lost: 100, survived: 0 }],
		},
		secondarySide: {
			role: 'defender', power: 1000,
			generals: [{ id: 'liubei', name: '刘备', level: 1 }],
			units: [{ unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 100, dispatched: 100, lost: 100, survived: 0 }],
		},
		rewards: { generalExp: 100, resources: {} },
		traits: [],
	}
	assert.equal(hasStandardUnitRows(detail.primarySide), true)
	assert.equal(hasStandardUnitRows(detail.secondarySide), true)
	assert.deepEqual(detail.primarySide.generals[0].traits.map((trait) => trait.traitId), ['weiwu_haoling', 'weiwu_tongyu'])
	assert.deepEqual(detail.primarySide.units[0], { unitType: 'huWei', unitName: '虎卫', amountBefore: 100, dispatched: 100, lost: 100, survived: 0 })
	assert.deepEqual(detail.secondarySide.units[0], { unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 100, dispatched: 100, lost: 100, survived: 0 })
	assert.deepEqual(detail.traits, [])
	assert.deepEqual(detail.rewards, { generalExp: 100, resources: {} })
})

test('攻守双方同一正式特性分别归属且旧结果弹窗使用真实特性 ID', () => {
	const options = { faction: 'shu', units: { shu: { shuInfantry: { name: '蜀步兵' } } } }
	const detail = {
		primarySide: {
			role: 'attacker', power: 10000,
			generals: [{ id: 'huangzhong', name: '黄忠', level: 1 }],
			units: [{ unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 1000, dispatched: 1000, lost: 600, survived: 400 }],
		},
		secondarySide: {
			role: 'defender', power: 10000,
			generals: [{ id: 'huangzhong', name: '黄忠', level: 1 }],
			units: [{ unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 1000, dispatched: 1000, lost: 600, survived: 400 }],
		},
		traits: [
			{
				traitId: 'laodang_yizhuang', traitName: '老当益壮', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'huangzhong',
				detail: { effectRate: 0.1, extraLosses: { shuInfantry: 100 }, triggerChance: 1 },
			},
			{
				traitId: 'laodang_yizhuang', traitName: '老当益壮', ownerSide: 'secondary', ownerRole: 'defender', generalId: 'huangzhong',
				detail: { effectRate: 0.1, extraLosses: { shuInfantry: 100 }, triggerChance: 1 },
			},
		],
	}
	assert.equal(hasStandardUnitRows(detail.primarySide), true)
	assert.equal(hasStandardUnitRows(detail.secondarySide), true)
	assert.deepEqual(detail.primarySide.units[0], { unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 1000, dispatched: 1000, lost: 600, survived: 400 })
	assert.deepEqual(detail.secondarySide.units[0], { unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 1000, dispatched: 1000, lost: 600, survived: 400 })
	assert.deepEqual(detail.traits.map((trait) => trait.traitId), ['laodang_yizhuang', 'laodang_yizhuang'])
	assert.deepEqual(detail.traits.map((trait) => resolveReportTraitDisplaySide({ sourceType: 'player_city', viewType: 'attack' }, trait)), ['primary', 'secondary'])
	for (const trait of detail.traits) {
		assert.equal(formatTraitOutcomeDetail('effectRate', trait.detail.effectRate), '设计效果比例: 10%')
		assert.equal(formatTraitOutcomeDetail('extraLosses', trait.detail.extraLosses, options), '追加损失: 蜀步兵 +100')
	}
	const source = readFileSync(new URL('../src/pages/map/components/BattleResultModal.tsx', import.meta.url), 'utf8')
	assert.match(source, /const outcome = report\.traitOutcomes\?\.\[traitId\]/)
	assert.match(source, /const displayTraitId = outcome\?\.traitId \|\| traitId/)
	assert.match(source, /getTraitMeta\(displayTraitId\)/)
})

test('攻守双方曹操的同一战前特性分别展示真实攻防修正', () => {
	const options = { faction: 'wei', units: { wei: { huWei: { name: '虎卫' } } } }
	const detail = {
		primarySide: {
			role: 'attacker', power: 1100,
			generals: [{ id: 'caocao', name: '曹操', level: 1 }],
			units: [{ unitType: 'huWei', unitName: '虎卫', amountBefore: 100, dispatched: 100, lost: 50, survived: 50 }],
		},
		secondarySide: {
			role: 'defender', power: 1100,
			generals: [{ id: 'caocao', name: '曹操', level: 1 }],
			units: [{ unitType: 'huWei', unitName: '虎卫', amountBefore: 100, dispatched: 100, lost: 50, survived: 50 }],
		},
		traits: [
			{
				traitId: 'weiwu_tongyu', traitName: '魏武统御', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'caocao',
				detail: { attackBonusRate: 0.1, defenseBonusRate: 0.1, attackModifiedUnits: { huWei: 1 }, infantryDefenseModifiedUnits: { huWei: 1 }, cavalryDefenseModifiedUnits: { huWei: 1 }, triggerChance: 1 },
			},
			{
				traitId: 'weiwu_tongyu', traitName: '魏武统御', ownerSide: 'secondary', ownerRole: 'defender', generalId: 'caocao',
				detail: { attackBonusRate: 0.1, defenseBonusRate: 0.1, attackModifiedUnits: { huWei: 1 }, infantryDefenseModifiedUnits: { huWei: 1 }, cavalryDefenseModifiedUnits: { huWei: 1 }, triggerChance: 1 },
			},
		],
	}
	assert.deepEqual(detail.traits.map((trait) => resolveReportTraitDisplaySide({ sourceType: 'player_city', viewType: 'attack' }, trait)), ['primary', 'secondary'])
	assert.deepEqual(detail.primarySide.units[0], { unitType: 'huWei', unitName: '虎卫', amountBefore: 100, dispatched: 100, lost: 50, survived: 50 })
	assert.deepEqual(detail.secondarySide.units[0], { unitType: 'huWei', unitName: '虎卫', amountBefore: 100, dispatched: 100, lost: 50, survived: 50 })
	for (const trait of detail.traits) {
		assert.equal(formatTraitOutcomeDetail('attackBonusRate', trait.detail.attackBonusRate), '设计攻击加成: 10%')
		assert.equal(formatTraitOutcomeDetail('defenseBonusRate', trait.detail.defenseBonusRate), '设计防御加成: 10%')
		assert.equal(formatTraitOutcomeDetail('attackModifiedUnits', trait.detail.attackModifiedUnits, options), '实际攻击修正: 虎卫 +1')
		assert.equal(formatTraitOutcomeDetail('infantryDefenseModifiedUnits', trait.detail.infantryDefenseModifiedUnits, options), '实际步防修正: 虎卫 +1')
		assert.equal(formatTraitOutcomeDetail('cavalryDefenseModifiedUnits', trait.detail.cavalryDefenseModifiedUnits, options), '实际骑防修正: 虎卫 +1')
	}
})

test('曹操进攻加成与孙权防守加成交叉展示真实战力兵损和归属', () => {
	const weiOptions = { faction: 'wei', units: { wei: { huWei: { name: '虎卫' } } } }
	const wuOptions = { faction: 'wu', units: { wu: { wuInfantry: { name: '吴步兵' } } } }
	const detail = {
		primarySide: {
			role: 'attacker', power: 11000, generals: [{ id: 'caocao', name: '曹操', level: 1 }],
			units: [{ unitType: 'huWei', unitName: '虎卫', amountBefore: 1000, dispatched: 1000, lost: 608, survived: 392 }],
		},
		secondarySide: {
			role: 'defender', power: 15000, generals: [{ id: 'sunquan', name: '孙权', level: 1 }],
			units: [{ unitType: 'wuInfantry', unitName: '吴步兵', amountBefore: 1000, dispatched: 1000, lost: 391, survived: 609 }],
		},
		traits: [
			{
				traitId: 'weiwu_tongyu', traitName: '魏武统御', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'caocao',
				detail: { attackBonusRate: 0.1, defenseBonusRate: 0.1, attackModifiedUnits: { huWei: 1 }, infantryDefenseModifiedUnits: { huWei: 1 }, cavalryDefenseModifiedUnits: { huWei: 1 }, triggerChance: 1 },
			},
			{
				traitId: 'jiangdong_gushou', traitName: '江东固守', ownerSide: 'secondary', ownerRole: 'defender', generalId: 'sunquan',
				detail: { defenseBonusRate: 0.5, infantryDefenseModifiedUnits: { wuInfantry: 5 }, cavalryDefenseModifiedUnits: { wuInfantry: 4 }, triggerChance: 1 },
			},
		],
	}
	assert.deepEqual(detail.traits.map((trait) => resolveReportTraitDisplaySide({ sourceType: 'player_city', viewType: 'attack' }, trait)), ['primary', 'secondary'])
	assert.deepEqual(detail.primarySide.units[0], { unitType: 'huWei', unitName: '虎卫', amountBefore: 1000, dispatched: 1000, lost: 608, survived: 392 })
	assert.deepEqual(detail.secondarySide.units[0], { unitType: 'wuInfantry', unitName: '吴步兵', amountBefore: 1000, dispatched: 1000, lost: 391, survived: 609 })
	assert.equal(formatTraitOutcomeDetail('attackModifiedUnits', detail.traits[0].detail.attackModifiedUnits, weiOptions), '实际攻击修正: 虎卫 +1')
	assert.equal(formatTraitOutcomeDetail('infantryDefenseModifiedUnits', detail.traits[1].detail.infantryDefenseModifiedUnits, wuOptions), '实际步防修正: 吴步兵 +5')
	assert.equal(formatTraitOutcomeDetail('cavalryDefenseModifiedUnits', detail.traits[1].detail.cavalryDefenseModifiedUnits, wuOptions), '实际骑防修正: 吴步兵 +4')
})

test('甄宓破防后孙权按当前整数防御加防并分别展示实际变化', () => {
	const options = { faction: 'wu', units: { wu: { wuInfantry: { name: '吴步兵' } } } }
	const detail = {
		sourceType: 'player_city', viewType: 'attack', battleType: 'plunder',
		primarySide: {
			role: 'attacker', power: 10000, generals: [{ id: 'zhenmi', name: '甄宓', level: 1 }],
			units: [{ unitType: 'huWei', unitName: '虎卫', amountBefore: 1000, dispatched: 1000, lost: 564, survived: 436 }],
		},
		secondarySide: {
			role: 'defender', power: 12000, generals: [{ id: 'sunquan', name: '孙权', level: 1 }],
			units: [{ unitType: 'wuInfantry', unitName: '吴步兵', amountBefore: 1000, dispatched: 1000, lost: 435, survived: 565 }],
		},
		rewards: { generalExp: 435, resources: {} },
		traits: [
			{
				traitId: 'meihuo_raozhen', traitName: '魅惑扰阵', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'zhenmi',
				detail: { enemyDefenseReductionRate: 0.25, infantryDefenseModifiedUnits: { wuInfantry: -2 }, cavalryDefenseModifiedUnits: { wuInfantry: -2 }, triggerChance: 1 },
			},
			{
				traitId: 'jiangdong_gushou', traitName: '江东固守', ownerSide: 'secondary', ownerRole: 'defender', generalId: 'sunquan',
				detail: { defenseBonusRate: 0.5, infantryDefenseModifiedUnits: { wuInfantry: 4 }, cavalryDefenseModifiedUnits: { wuInfantry: 3 }, triggerChance: 1 },
			},
		],
	}
	assert.deepEqual(detail.traits.map((trait) => trait.traitId), ['meihuo_raozhen', 'jiangdong_gushou'])
	assert.deepEqual(detail.traits.map((trait) => resolveReportTraitDisplaySide(detail, trait)), ['primary', 'secondary'])
	assert.deepEqual(detail.traits.map((trait) => getTraitMeta(trait.traitId).trigger), ['主动进攻战斗前', '防守/增援战斗前'])
	assert.deepEqual(detail.primarySide.units[0], { unitType: 'huWei', unitName: '虎卫', amountBefore: 1000, dispatched: 1000, lost: 564, survived: 436 })
	assert.deepEqual(detail.secondarySide.units[0], { unitType: 'wuInfantry', unitName: '吴步兵', amountBefore: 1000, dispatched: 1000, lost: 435, survived: 565 })
	assert.equal(detail.primarySide.power, 10000)
	assert.equal(detail.secondarySide.power, 12000)
	assert.equal(detail.rewards.generalExp, 435)
	assert.equal(formatTraitOutcomeDetail('enemyDefenseReductionRate', detail.traits[0].detail.enemyDefenseReductionRate, options), '设计敌方防御降低: 25%')
	assert.equal(formatTraitOutcomeDetail('infantryDefenseModifiedUnits', detail.traits[0].detail.infantryDefenseModifiedUnits, options), '实际步防修正: 吴步兵 -2')
	assert.equal(formatTraitOutcomeDetail('cavalryDefenseModifiedUnits', detail.traits[0].detail.cavalryDefenseModifiedUnits, options), '实际骑防修正: 吴步兵 -2')
	assert.equal(formatTraitOutcomeDetail('defenseBonusRate', detail.traits[1].detail.defenseBonusRate, options), '设计防御加成: 50%')
	assert.equal(formatTraitOutcomeDetail('infantryDefenseModifiedUnits', detail.traits[1].detail.infantryDefenseModifiedUnits, options), '实际步防修正: 吴步兵 +4')
	assert.equal(formatTraitOutcomeDetail('cavalryDefenseModifiedUnits', detail.traits[1].detail.cavalryDefenseModifiedUnits, options), '实际骑防修正: 吴步兵 +3')
})

test('攻守双方刘备仁主守护分别展示原始阵亡和最终存活', () => {
	const options = { faction: 'shu', units: { shu: { shuInfantry: { name: '蜀步兵' } } } }
	const detail = {
		primarySide: {
			role: 'attacker', power: 10000,
			generals: [{ id: 'liubei', name: '刘备', level: 1 }],
			units: [{ unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 1000, dispatched: 1000, lost: 500, survived: 675 }],
		},
		secondarySide: {
			role: 'defender', power: 10000,
			generals: [{ id: 'liubei', name: '刘备', level: 1 }],
			units: [{ unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 1000, dispatched: 1000, lost: 500, survived: 675 }],
		},
		traits: [
			{
				traitId: 'renzhu_shouhu', traitName: '仁主守护', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'liubei',
				detail: { effectRate: 0.35, revivedUnits: { shuInfantry: 175 }, triggerChance: 0.6 },
			},
			{
				traitId: 'renzhu_shouhu', traitName: '仁主守护', ownerSide: 'secondary', ownerRole: 'defender', generalId: 'liubei',
				detail: { effectRate: 0.35, revivedUnits: { shuInfantry: 175 }, triggerChance: 0.6 },
			},
		],
	}
	assert.deepEqual(detail.primarySide.units[0], { unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 1000, dispatched: 1000, lost: 500, survived: 675 })
	assert.deepEqual(detail.secondarySide.units[0], { unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 1000, dispatched: 1000, lost: 500, survived: 675 })
	assert.deepEqual(detail.traits.map((trait) => trait.traitId), ['renzhu_shouhu', 'renzhu_shouhu'])
	assert.equal(new Set(detail.traits.map((trait) => `${trait.ownerSide}:${trait.traitId}`)).size, 2)
	assert.deepEqual(detail.traits.map((trait) => resolveReportTraitDisplaySide({ sourceType: 'player_city', viewType: 'attack' }, trait)), ['primary', 'secondary'])
	for (const trait of detail.traits) {
		assert.equal(formatTraitOutcomeDetail('revivedUnits', trait.detail.revivedUnits, options), '复活兵力: 蜀步兵 +175')
	}
})

test('刘备多兵种大额阵亡逐兵种按百分比复活且不设人数上限', () => {
	const options = { faction: 'shu', units: { shu: { shuCavalry: { name: '蜀骑兵' }, shuInfantry: { name: '蜀步兵' } } } }
	const detail = {
		primarySide: {
			role: 'attacker', power: 720000,
			generals: [{ id: 'liubei', name: '刘备', level: 1 }],
			units: [
				{ unitType: 'shuCavalry', unitName: '蜀骑兵', amountBefore: 30000, dispatched: 30000, lost: 30000, survived: 10500 },
				{ unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 30000, dispatched: 30000, lost: 30000, survived: 10500 },
			],
		},
		secondarySide: {
			role: 'defender', power: 8833333,
			generals: [{ id: 'caocao', name: '曹操', level: 1 }],
			units: [{ unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 1000000, dispatched: 1000000, lost: 28296, survived: 971704 }],
		},
		rewards: { generalExp: 28296, resources: {} },
		traits: [
			{
				traitId: 'renzhu_shouhu', traitName: '仁主守护', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'liubei',
				detail: { effectRate: 0.35, revivedUnits: { shuCavalry: 10500, shuInfantry: 10500 }, totalRevived: 21000, triggerChance: 0.6 },
			},
		],
	}
	assert.deepEqual(detail.primarySide.units, [
		{ unitType: 'shuCavalry', unitName: '蜀骑兵', amountBefore: 30000, dispatched: 30000, lost: 30000, survived: 10500 },
		{ unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 30000, dispatched: 30000, lost: 30000, survived: 10500 },
	])
	assert.deepEqual(detail.traits.map((trait) => resolveReportTraitDisplaySide({ sourceType: 'player_city', viewType: 'attack' }, trait)), ['primary'])
	assert.equal(formatTraitOutcomeDetail('effectRate', detail.traits[0].detail.effectRate), '设计效果比例: 35%')
	assert.equal(formatTraitOutcomeDetail('revivedUnits', detail.traits[0].detail.revivedUnits, options), '复活兵力: 蜀骑兵 +10,500、蜀步兵 +10,500')
	assert.equal(formatTraitOutcomeDetail('totalRevived', detail.traits[0].detail.totalRevived), '复活总数: 21,000')
})

test('七项战前攻击加成展示准确设计值、目标和实际修正', () => {
	const expected = {
		sizhandaodi: [/步兵/, /35%/, /仅在主动进攻/],
		weizhen_xiaoyao: [/骑兵/, /35%/, /仅在主动进攻/],
		wusheng_pojun: [/全军/, /20%/, /仅在主动进攻/],
		wanren_nuhou: [/步兵/, /20%/, /仅在主动进攻/],
		xiaobawang_tieqi: [/霸王骑/, /50 点/, /仅在主动进攻/],
		meizhoulang_junlue: [/全军/, /5%/, /仅在主动进攻/],
		jinfan_qixi: [/全军/, /10%/, /仅在掠夺战主动进攻/],
	} as const
	for (const [traitId, patterns] of Object.entries(expected)) {
		for (const pattern of patterns) assert.match(getTraitMeta(traitId).description, pattern, traitId)
	}
  assert.equal(formatTraitOutcomeDetail('attackBonusRate', 0.35), '设计攻击加成: 35%')
  assert.equal(formatTraitOutcomeDetail('unitAttackFlat', 50), '设计单位攻击增加: 50')
  for (const traitId of ['sizhandaodi', 'weizhen_xiaoyao', 'wusheng_pojun', 'wanren_nuhou', 'xiaobawang_tieqi', 'meizhoulang_junlue']) {
    assert.equal(getTraitMeta(traitId).trigger, '主动进攻战斗前')
  }
  assert.equal(getTraitMeta('jinfan_qixi').trigger, '掠夺战战斗前')
})

test('奇兵绕后明确限制为主动进攻并展示两类实际防御变化', () => {
	const trait = getTraitMeta('qibing_raohou')
	const options = { faction: 'wei', units: { wei: { weiInfantry: { name: '魏步兵' } } } }
	assert.equal(trait.trigger, '主动进攻战斗前')
	assert.match(trait.description, /仅在主动进攻/)
	assert.equal(formatTraitOutcomeDetail('infantryDefenseModifiedUnits', { weiInfantry: -2 }, options), '实际步防修正: 魏步兵 -2')
	assert.equal(formatTraitOutcomeDetail('cavalryDefenseModifiedUnits', { weiInfantry: -2 }, options), '实际骑防修正: 魏步兵 -2')
})

test('纯防御特性明确限制为防守或增援并展示正式数值', () => {
	for (const traitId of ['dunzhen_fangyu', 'gushou_hanzhong', 'jiangdong_gushou']) {
		const trait = getTraitMeta(traitId)
		assert.equal(trait.trigger, '防守/增援战斗前')
		assert.match(trait.description, /仅在防守或作为援军/)
	}
	assert.match(getTraitMeta('dunzhen_fangyu').description, /60% 概率/)
	assert.match(getTraitMeta('dunzhen_fangyu').description, /防御提升 30%/)
	assert.match(getTraitMeta('gushou_hanzhong').description, /20 点/)
	assert.match(getTraitMeta('jiangdong_gushou').description, /50% 概率/)
	assert.match(getTraitMeta('jiangdong_gushou').description, /防御提升 50%/)
	assert.equal(formatTraitOutcomeDetail('infantryDefenseModifiedUnits', { huWei: 4 }), '实际步防修正: 虎卫 +4')
	assert.equal(formatTraitOutcomeDetail('cavalryDefenseModifiedUnits', { huWei: 3 }), '实际骑防修正: 虎卫 +3')
	assert.equal(formatTraitOutcomeDetail('defenseBonusRate', 0.3), '设计防御加成: 30%')
	assert.equal(formatTraitOutcomeDetail('generalDefenseFlat', 20), '设计全军防御增加: 20')
})

test('孙权防守方向正确但江东固守未命中时不补造防御加成', () => {
	const detail = {
		sourceType: 'player_city', viewType: 'attack', battleType: 'plunder',
		primarySide: {
			role: 'attacker', power: 10000,
			units: [{ unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 1000, dispatched: 1000, lost: 500, survived: 500 }],
		},
		secondarySide: {
			role: 'defender', power: 10000,
			generals: [{ id: 'sunquan', name: '孙权', traits: [{ traitId: 'jiangdong_gushou', name: '江东固守', params: { triggerChance: 0.5, defenseBonusRate: 0.5 } }] }],
			units: [{ unitType: 'wuInfantry', unitName: '吴步兵', amountBefore: 1000, dispatched: 1000, lost: 500, survived: 500 }],
		},
		traits: [],
	}
	assert.equal(detail.secondarySide.generals[0].traits[0].traitId, 'jiangdong_gushou')
	assert.equal(hasTraitEntries(detail), false)
	assert.deepEqual(detail.primarySide, {
		role: 'attacker', power: 10000,
		units: [{ unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 1000, dispatched: 1000, lost: 500, survived: 500 }],
	})
	assert.equal(detail.secondarySide.power, 10000)
	assert.deepEqual(detail.secondarySide.units, [{ unitType: 'wuInfantry', unitName: '吴步兵', amountBefore: 1000, dispatched: 1000, lost: 500, survived: 500 }])
})

test('孙权作为援军时江东固守未命中只展示基础防御和完整损失', () => {
	const detail = {
		primarySide: {
			role: 'reinforcement', power: 1010,
			generals: [{ id: 'sunquan', name: '孙权', level: 1, traits: [{ traitId: 'jiangdong_gushou', name: '江东固守' }] }],
			units: [{ unitType: 'wuInfantry', unitName: '吴步兵', amountBefore: 100, dispatched: 100, lost: 100, survived: 0 }],
		},
		rewards: { generalExp: 97, resources: {} },
		traits: [],
	}
	assert.equal(hasStandardUnitRows(detail.primarySide), true)
	assert.equal(hasTraitEntries(detail), false)
	assert.deepEqual(detail.primarySide.generals[0].traits.map((trait) => trait.traitId), ['jiangdong_gushou'])
	assert.equal(detail.primarySide.power, 1010)
	assert.deepEqual(detail.primarySide.units[0], { unitType: 'wuInfantry', unitName: '吴步兵', amountBefore: 100, dispatched: 100, lost: 100, survived: 0 })
	assert.deepEqual(detail.rewards, { generalExp: 97, resources: {} })
})

test('龙胆救援只描述防守或增援的最终实际减损', () => {
	const trait = getTraitMeta('longdan_jiuyuan')
	const options = { faction: 'shu', units: { shu: { shuInfantry: { name: '蜀步兵' } } } }
	assert.equal(trait.trigger, '防守/增援战斗结算后')
	assert.match(trait.description, /仅在防守或增援/)
	assert.match(trait.description, /主动进攻无效/)
	assert.match(trait.description, /最终实际损失/)
	assert.equal(formatTraitOutcomeDetail('lossReductionRate', 0.2), '设计减损比例: 20%')
	assert.equal(formatTraitOutcomeDetail('reducedLosses', { shuInfantry: 20 }, options), '减少损失: 蜀步兵 +20')
})

test('主城赵云龙胆只减少主守军损失且不替同兵种援军减损', () => {
	const report = {
		id: 'main-longdan-source-isolation', playerId: 'attacker', ownerPlayerId: 'attacker', viewType: 'attack', sourceType: 'player_city',
		battleType: 'plunder', type: 'attack', result: 'draw', defenderRevealed: true, rewards: {}, read: true, createdAt: '2026-07-20T13:00:00Z',
		pvpReinforcements: [{
			reinforcementId: 'rein_liubei', fromPlayerId: 'helper', fromPlayerName: '刘备援军', faction: 'shu',
			troops: { shuInfantry: 500 }, generalExpGained: 500, generals: [{ id: 'liubei', name: '刘备', level: 1 }],
		}],
		pvpReinforcementLosses: { rein_liubei: { shuInfantry: 250 } },
		detail: {
			id: 'main-longdan-source-isolation', sourceType: 'player_city', viewType: 'attack', battleType: 'plunder', result: 'draw',
			primarySide: {
				role: 'attacker', power: 10000, generals: [{ id: 'caocao', name: '曹操', level: 1 }],
				units: [{ unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 1000, dispatched: 1000, lost: 500, survived: 500 }],
			},
			secondarySide: {
				role: 'defender', power: 10000, generals: [{ id: 'zhaoyun', name: '赵云', level: 1 }],
				units: [{ unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 500, dispatched: 500, lost: 200, survived: 300 }],
			},
			rewards: { generalExp: 450, resources: {} },
			traits: [{
				traitId: 'longdan_jiuyuan', traitName: '龙胆救援', ownerSide: 'secondary', ownerRole: 'defender', generalId: 'zhaoyun',
				detail: { lossReductionRate: 0.2, reducedLosses: { shuInfantry: 50 }, triggerChance: 1 },
			}],
		},
	}
	const detail = normalizeBattleReportDetail(report)
	const pvp = detail.extra?.pvp && typeof detail.extra.pvp === 'object' ? detail.extra.pvp : undefined
	const reinforcement = pvp?.reinforcements[0]
	const options = { faction: 'shu', units: { shu: { shuInfantry: { name: '蜀步兵' } } } }

	assert.deepEqual(detail.primarySide.units[0], { unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 1000, dispatched: 1000, lost: 500, survived: 500 })
	assert.deepEqual(detail.secondarySide?.units[0], { unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 500, dispatched: 500, lost: 200, survived: 300 })
	assert.equal(detail.rewards?.generalExp, 450)
	assert.deepEqual(reinforcement?.troops, { shuInfantry: 500 })
	assert.equal(reinforcement?.generalExpGained, 500)
	assert.deepEqual(pvp?.reinforcementLosses, { rein_liubei: { shuInfantry: 250 } })
	assert.deepEqual(detail.traits?.map((trait) => trait.traitId), ['longdan_jiuyuan'])
	assert.equal(resolveReportTraitDisplaySide(detail, detail.traits?.[0]), 'secondary')
	assert.equal(formatTraitOutcomeDetail('reducedLosses', detail.traits?.[0].detail.reducedLosses, options), '减少损失: 蜀步兵 +50')
})

test('主城赵云龙胆合法未命中时主守军和同兵种援军都承担完整损失', () => {
	const report = {
		id: 'main-longdan-probability-miss', playerId: 'attacker', ownerPlayerId: 'attacker', viewType: 'attack', sourceType: 'player_city',
		battleType: 'plunder', type: 'attack', result: 'draw', defenderRevealed: true, rewards: {}, read: true, createdAt: '2026-07-20T15:00:00Z',
		pvpReinforcements: [{
			reinforcementId: 'rein_liubei_miss', fromPlayerId: 'helper', fromPlayerName: '刘备援军', faction: 'shu',
			troops: { shuInfantry: 500 }, generalExpGained: 500, generals: [{ id: 'liubei', name: '刘备', level: 1 }],
		}],
		pvpReinforcementLosses: { rein_liubei_miss: { shuInfantry: 250 } },
		detail: {
			id: 'main-longdan-probability-miss', sourceType: 'player_city', viewType: 'attack', battleType: 'plunder', result: 'draw',
			primarySide: {
				role: 'attacker', power: 10000, generals: [{ id: 'caocao', name: '曹操', level: 1 }],
				units: [{ unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 1000, dispatched: 1000, lost: 500, survived: 500 }],
			},
			secondarySide: {
				role: 'defender', power: 10000,
				generals: [{ id: 'zhaoyun', name: '赵云', level: 1, traits: [{ traitId: 'longdan_jiuyuan', name: '龙胆救援' }] }],
				units: [{ unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 500, dispatched: 500, lost: 250, survived: 250 }],
			},
			rewards: { generalExp: 500, resources: {} }, traits: [],
		},
	}
	const detail = normalizeBattleReportDetail(report)
	const pvp = detail.extra?.pvp && typeof detail.extra.pvp === 'object' ? detail.extra.pvp : undefined
	assert.equal(hasTraitEntries(detail), false)
	assert.deepEqual(detail.secondarySide?.generals?.[0]?.traits?.map((trait) => trait.traitId), ['longdan_jiuyuan'])
	assert.deepEqual(detail.primarySide.units[0], { unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 1000, dispatched: 1000, lost: 500, survived: 500 })
	assert.deepEqual(detail.secondarySide?.units[0], { unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 500, dispatched: 500, lost: 250, survived: 250 })
	assert.equal(detail.rewards?.generalExp, 500)
	assert.deepEqual(pvp?.reinforcementLosses, { rein_liubei_miss: { shuInfantry: 250 } })
	assert.equal(pvp?.reinforcements[0]?.generalExpGained, 500)
})

test('主城刘备只复活主守军且同兵种援军保持原始阵亡', () => {
	const report = {
		id: 'main-liubei-source-isolation', playerId: 'attacker', ownerPlayerId: 'attacker', viewType: 'attack', sourceType: 'player_city',
		battleType: 'plunder', type: 'attack', result: 'draw', defenderRevealed: true, rewards: {}, read: true, createdAt: '2026-07-20T14:00:00Z',
		pvpReinforcements: [{
			reinforcementId: 'rein_guanyu', fromPlayerId: 'helper', fromPlayerName: '关羽援军', faction: 'shu',
			troops: { shuInfantry: 500 }, generalExpGained: 500, generals: [{ id: 'guanyu', name: '关羽', level: 1 }],
		}],
		pvpReinforcementLosses: { rein_guanyu: { shuInfantry: 250 } },
		detail: {
			id: 'main-liubei-source-isolation', sourceType: 'player_city', viewType: 'attack', battleType: 'plunder', result: 'draw',
			primarySide: {
				role: 'attacker', power: 10000, generals: [{ id: 'caocao', name: '曹操', level: 1 }],
				units: [{ unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 1000, dispatched: 1000, lost: 500, survived: 500 }],
			},
			secondarySide: {
				role: 'defender', power: 10000, generals: [{ id: 'liubei', name: '刘备', level: 1 }],
				units: [{ unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 500, dispatched: 500, lost: 250, survived: 337 }],
			},
			rewards: { generalExp: 500, resources: {} },
			traits: [
				{
					traitId: 'renzhu_shouhu', traitName: '仁主守护', ownerSide: 'secondary', ownerRole: 'defender', generalId: 'liubei',
					detail: { effectRate: 0.35, revivedUnits: { shuInfantry: 87 }, triggerChance: 0.6 },
				},
			],
		},
	}
	const detail = normalizeBattleReportDetail(report)
	const pvp = detail.extra?.pvp && typeof detail.extra.pvp === 'object' ? detail.extra.pvp : undefined
	const reinforcement = pvp?.reinforcements[0]
	const options = { faction: 'shu', units: { shu: { shuInfantry: { name: '蜀步兵' } } } }

	assert.deepEqual(detail.primarySide.units[0], { unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 1000, dispatched: 1000, lost: 500, survived: 500 })
	assert.deepEqual(detail.secondarySide?.units[0], { unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 500, dispatched: 500, lost: 250, survived: 337 })
	assert.equal(detail.rewards?.generalExp, 500)
	assert.deepEqual(reinforcement?.troops, { shuInfantry: 500 })
	assert.equal(reinforcement?.generalExpGained, 500)
	assert.deepEqual(pvp?.reinforcementLosses, { rein_guanyu: { shuInfantry: 250 } })
	assert.deepEqual(detail.traits?.map((trait) => trait.traitId), ['renzhu_shouhu'])
	assert.deepEqual(detail.traits?.map((trait) => resolveReportTraitDisplaySide(detail, trait)), ['secondary'])
	assert.equal(formatTraitOutcomeDetail('revivedUnits', detail.traits?.[0].detail.revivedUnits, options), '复活兵力: 蜀步兵 +87')
})

test('刘备被动属性与概率复活使用独立时间线和当前参数', () => {
	const options = { faction: 'shu', units: { shu: { shuInfantry: { name: '蜀步兵' } } } }
	const rende = getTraitMeta('rende')
	const guard = getTraitMeta('renzhu_shouhu')
	assert.equal(rende.trigger, '永久被动')
	assert.equal(guard.trigger, '进攻/防守/增援战斗结束后')
	assert.match(rende.description, /固定增加 10 点内政/)
	assert.match(rende.description, /12 点统率/)
	assert.match(guard.description, /60%/)
	assert.match(guard.description, /真实阵亡/)
	assert.match(guard.description, /35%/)
	assert.doesNotMatch(guard.description, /上限|返还/)
	assert.equal(formatTraitOutcomeDetail('triggerChance', 0.6), '触发概率: 60%')
	assert.equal(formatTraitOutcomeDetail('revivedUnits', { shuInfantry: 35 }, options), '复活兵力: 蜀步兵 +35')
})

test('郭嘉在进攻、防守或增援战后按真实阵亡复活', () => {
	const trait = getTraitMeta('guicai_yice')
	assert.equal(trait.trigger, '进攻/防守/增援战斗结束后')
	assert.match(trait.description, /本场真实阵亡/)
	assert.match(trait.description, /22%/)
	assert.match(trait.description, /逐兵种复活/)
	assert.equal(formatTraitOutcomeDetail('actualLostUnits', { weiInfantry: 100 }), '本场真实阵亡: weiInfantry +100')
	assert.equal(formatTraitOutcomeDetail('revivedUnits', { weiInfantry: 22 }), '复活兵力: weiInfantry +22')
})

test('郭嘉在 PVP 平局中的真实阵亡、复活和最终兵力完整展示', () => {
	const report = {
		id: 'draw-guojia', playerId: 'attacker', ownerPlayerId: 'attacker', viewType: 'attack', sourceType: 'player_city',
		battleType: 'plunder', type: 'attack', result: 'draw', defenderRevealed: true, rewards: {}, read: true, createdAt: '2026-07-20T15:00:00Z',
		detail: {
			id: 'draw-guojia', sourceType: 'player_city', viewType: 'attack', battleType: 'plunder', result: 'draw', winnerSide: 'none',
			primarySide: {
				role: 'attacker', faction: 'wei', power: 10000,
				generals: [{ id: 'guojia', name: '郭嘉', level: 1, traits: [{ traitId: 'guicai_yice', name: '鬼才遗策' }] }],
				units: [{ unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 1000, dispatched: 1000, lost: 500, survived: 610 }],
			},
			secondarySide: {
				role: 'defender', faction: 'shu', power: 10000,
				units: [{ unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 1000, dispatched: 1000, lost: 500, survived: 500 }],
			},
			rewards: { generalExp: 500, resources: {} },
			traits: [{
				traitId: 'guicai_yice', traitName: '鬼才遗策', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'guojia',
				detail: { effectRate: 0.22, actualLostUnits: { weiInfantry: 500 }, revivedUnits: { weiInfantry: 110 }, totalRevived: 110, triggerChance: 1 },
			}],
		},
	}
	const detail = normalizeBattleReportDetail(report)
	assert.equal(detail.primarySide.generals?.[0].traits?.[0].traitId, 'guicai_yice')
	assert.equal(hasTraitEntries(detail), true)
	assert.deepEqual(detail.primarySide.units[0], { unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 1000, dispatched: 1000, lost: 500, survived: 610 })
	assert.equal(formatTraitOutcomeDetail('actualLostUnits', detail.traits[0].detail.actualLostUnits), '本场真实阵亡: weiInfantry +500')
	assert.equal(formatTraitOutcomeDetail('revivedUnits', detail.traits[0].detail.revivedUnits), '复活兵力: weiInfantry +110')
	assert.equal(detail.rewards?.generalExp, 500)
})

test('郭嘉在 NPC 与黄巾平局中都展示战后复活', () => {
	for (const sourceType of ['npc_city', 'yellow_turban']) {
		const detail = {
			sourceType, viewType: sourceType === 'yellow_turban' ? 'defense' : 'attack', battleType: sourceType, result: 'draw', winnerSide: 'none',
			primarySide: {
				role: 'attacker', faction: 'wei', power: 1000,
				units: [{ unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 100, dispatched: 100, lost: 100, survived: sourceType === 'npc_city' ? 22 : 0 }],
			},
			secondarySide: {
				role: 'defender', faction: 'wei', power: 1000,
				generals: sourceType === 'yellow_turban' ? [{ id: 'guojia', name: '郭嘉', level: 1 }] : [],
				units: [{ unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 100, dispatched: 100, lost: 100, survived: sourceType === 'yellow_turban' ? 22 : 0 }],
			},
			rewards: { generalExp: 100, resources: {} },
			traits: [{
				traitId: 'guicai_yice', traitName: '鬼才遗策',
				ownerSide: sourceType === 'yellow_turban' ? 'secondary' : 'primary',
				ownerRole: sourceType === 'yellow_turban' ? 'defender' : 'attacker',
				generalId: 'guojia',
				detail: { effectRate: 0.22, actualLostUnits: { weiInfantry: 100 }, revivedUnits: { weiInfantry: 22 }, totalRevived: 22 },
			}],
		}
		assert.equal(hasTraitEntries(detail), true)
		assert.equal(detail.traits[0].detail.revivedUnits.weiInfantry, 22)
		assert.equal(detail.rewards.generalExp, 100)
	}
})

test('七进七出是出征或增援过程能力且前端使用后端到达时间', () => {
	const trait = getTraitMeta('qijin_qichu')
	assert.equal(trait.trigger, '行军创建时')
	assert.match(trait.description, /主动出征或增援/)
	assert.match(trait.description, /固定提升 100%/)
	assert.match(trait.description, /最低 60 秒/)
	assert.match(trait.description, /不作为战斗触发特性/)
	const source = readFileSync(new URL('../src/components/MarchAlertTags.tsx', import.meta.url), 'utf8')
	assert.match(source, /endsAt: returning \? march\.returnsAt : march\.arrivesAt/)
	assert.doesNotMatch(source, /qijin_qichu|speedBonusRate/)
})

test('疾行奔袭与虎虎生威明确为指定骑兵永久被动，其他正式行军特性保留概率倍率和最低时长', () => {
	const jixing = getTraitMeta('jixing_benxi')
	assert.equal(jixing.trigger, '永久被动')
	assert.match(jixing.description, /骁骑营/)
	assert.match(jixing.description, /18 点攻击/)
	assert.match(jixing.description, /5 点移动/)
	assert.match(jixing.description, /不作为战斗触发特性/)
	const huhu = getTraitMeta('huhu_shengwei')
	assert.equal(huhu.trigger, '永久被动')
	assert.match(huhu.description, /虎豹骑/)
	assert.match(huhu.description, /12 点攻击/)
	assert.match(huhu.description, /5 点移动/)
	assert.match(huhu.description, /进攻、防守、增援均持续生效/)
	assert.match(huhu.description, /不作为战斗触发特性/)
	const detailSource = readFileSync(new URL('../src/pages/news/components/BattleReportDetail.tsx', import.meta.url), 'utf8')
	assert.match(detailSource, /huhu_shengwei/)
	assert.match(detailSource, /虎虎生威/)
	assert.match(detailSource, /虎豹骑/)

	const dujiang = getTraitMeta('baiyi_dujiang')
	assert.match(dujiang.description, /35% 概率/)
	assert.match(dujiang.description, /20%/)
	assert.match(dujiang.description, /最低 60 秒/)

	const jixingBonus = getTraitMeta('baiyi_jixing')
	assert.match(jixingBonus.description, /固定提升 20%/)
	assert.match(jixingBonus.description, /逐次叠加/)

	const lightning = getTraitMeta('kuairu_shandian')
	assert.match(lightning.description, /35% 概率/)
	assert.match(lightning.description, /400%/)
	assert.match(lightning.description, /最低 30 秒/)
})

test('太史慈主动进攻只使用快如闪电且信义勇烈明确限于援军', () => {
	const lightning = getTraitMeta('kuairu_shandian')
	const valor = getTraitMeta('xinyi_yonglie')
	const options = { faction: 'wu', units: { wu: { wuInfantry: { name: '吴步兵' } } } }
	assert.equal(lightning.trigger, '行军创建时')
	assert.match(lightning.description, /不作为战斗触发特性/)
	assert.equal(valor.trigger, '援军战斗前')
	assert.match(valor.description, /援军自身 10% 步兵、骑兵防御/)
	assert.match(valor.description, /主动进攻或主城守军无效/)
	assert.equal(formatTraitOutcomeDetail('defenseBonusRate', 0.1), '设计防御加成: 10%')
	assert.equal(formatTraitOutcomeDetail('infantryDefenseModifiedUnits', { wuInfantry: 1 }, options), '实际步防修正: 吴步兵 +1')
	assert.equal(formatTraitOutcomeDetail('cavalryDefenseModifiedUnits', { wuInfantry: 1 }, options), '实际骑防修正: 吴步兵 +1')
})

test('快如闪电合法未命中时保持基线行军且信义勇烈正常加防', () => {
	const march = {
		durationSeconds: 2970,
		speedMultiplier: 3000 / 2970,
		startedAt: '2026-07-20T00:00:00Z',
		arrivesAt: '2026-07-20T00:49:30Z',
	}
	assert.equal((new Date(march.arrivesAt).getTime() - new Date(march.startedAt).getTime()) / 1000, march.durationSeconds)
	assert.equal(march.durationSeconds, 2970)
	assert.notEqual(march.durationSeconds, 594)

	const detail = {
		primarySide: {
			role: 'reinforcement', power: 1110,
			generals: [{
				id: 'taishici', name: '太史慈', level: 1,
				traits: [{ traitId: 'kuairu_shandian', name: '快如闪电' }, { traitId: 'xinyi_yonglie', name: '信义勇烈' }],
			}],
			units: [{ unitType: 'wuInfantry', unitName: '吴步兵', amountBefore: 100, dispatched: 100, lost: 98, survived: 2 }],
		},
		secondarySide: {
			role: 'attacker', power: 1100,
			units: [{ unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 110, dispatched: 110, lost: 110, survived: 0 }],
		},
		rewards: { generalExp: 110, resources: {} },
		traits: [{
			traitId: 'xinyi_yonglie', traitName: '信义勇烈', ownerSide: 'reinforcement', generalId: 'taishici',
			detail: { defenseBonusRate: 0.1, infantryDefenseModifiedUnits: { wuInfantry: 1 }, cavalryDefenseModifiedUnits: { wuInfantry: 1 } },
		}],
	}
	const options = { faction: 'wu', units: { wu: { wuInfantry: { name: '吴步兵' } } } }
	assert.equal(hasStandardUnitRows(detail.primarySide), true)
	assert.equal(hasStandardUnitRows(detail.secondarySide), true)
	assert.equal(hasTraitEntries(detail), true)
	assert.deepEqual(detail.traits.map((trait) => trait.traitId), ['xinyi_yonglie'])
	assert.deepEqual(detail.primarySide.generals[0].traits.map((trait) => trait.traitId), ['kuairu_shandian', 'xinyi_yonglie'])
	assert.deepEqual(detail.primarySide.units[0], { unitType: 'wuInfantry', unitName: '吴步兵', amountBefore: 100, dispatched: 100, lost: 98, survived: 2 })
	assert.deepEqual(detail.secondarySide.units[0], { unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 110, dispatched: 110, lost: 110, survived: 0 })
	assert.equal(detail.primarySide.power, 1110)
	assert.equal(detail.rewards.generalExp, 110)
	assert.equal(formatTraitOutcomeDetail('defenseBonusRate', detail.traits[0].detail.defenseBonusRate), '设计防御加成: 10%')
	assert.equal(formatTraitOutcomeDetail('infantryDefenseModifiedUnits', detail.traits[0].detail.infantryDefenseModifiedUnits, options), '实际步防修正: 吴步兵 +1')
	assert.equal(formatTraitOutcomeDetail('cavalryDefenseModifiedUnits', detail.traits[0].detail.cavalryDefenseModifiedUnits, options), '实际骑防修正: 吴步兵 +1')
})

test('吕蒙双行军特性只作用到达时间且不会被战报补造成触发', () => {
	for (const traitId of ['baiyi_dujiang', 'baiyi_jixing']) {
		const trait = getTraitMeta(traitId)
		assert.equal(trait.trigger, '行军创建时')
		assert.match(trait.description, /不作为战斗触发特性|逐次叠加/)
	}
	const detailSource = readFileSync(new URL('../src/pages/news/components/BattleReportDetail.tsx', import.meta.url), 'utf8')
	assert.match(detailSource, /const visible = \(detail\.traits \?\? \[\]\)\.filter/)
	assert.doesNotMatch(detailSource, /const visible = \(side\.generals/)
})

test('白衣渡江合法未命中时只采用白衣急行后的行军与战报结果', () => {
	const march = {
		durationSeconds: 2475,
		speedMultiplier: 3000 / 2475,
		startedAt: '2026-07-20T00:00:00Z',
		arrivesAt: '2026-07-20T00:41:15Z',
		returnStartedAt: '2026-07-20T01:00:00Z',
		returnsAt: '2026-07-20T01:41:15Z',
	}
	assert.equal((new Date(march.arrivesAt).getTime() - new Date(march.startedAt).getTime()) / 1000, march.durationSeconds)
	assert.equal((new Date(march.returnsAt).getTime() - new Date(march.returnStartedAt).getTime()) / 1000, march.durationSeconds)
	assert.equal(Math.abs(march.speedMultiplier - 1.2121212121212122) < 1e-12, true)

	const detail = {
		primarySide: {
			role: 'attacker', power: 1000,
			generals: [{
				id: 'lvmeng', name: '吕蒙', level: 1,
				traits: [{ traitId: 'baiyi_dujiang', name: '白衣渡江' }, { traitId: 'baiyi_jixing', name: '白衣急行' }],
			}],
			units: [{ unitType: 'wuInfantry', unitName: '吴步兵', amountBefore: 100, dispatched: 100, lost: 0, survived: 100 }],
		},
		secondarySide: {
			role: 'defender', power: 10,
			units: [{ unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 1, dispatched: 1, lost: 1, survived: 0 }],
		},
		rewards: { generalExp: 1, resources: {} },
		traits: [],
	}
	assert.equal(hasStandardUnitRows(detail.primarySide), true)
	assert.equal(hasStandardUnitRows(detail.secondarySide), true)
	assert.equal(hasTraitEntries(detail), false)
	assert.deepEqual(detail.primarySide.generals[0].traits.map((trait) => trait.traitId), ['baiyi_dujiang', 'baiyi_jixing'])
	assert.deepEqual(detail.primarySide.units[0], { unitType: 'wuInfantry', unitName: '吴步兵', amountBefore: 100, dispatched: 100, lost: 0, survived: 100 })
	assert.deepEqual(detail.secondarySide.units[0], { unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 1, dispatched: 1, lost: 1, survived: 0 })
	assert.deepEqual(detail.rewards, { generalExp: 1, resources: {} })
})

test('吕蒙作为援军时拥有快照不能成为触发效果来源', () => {
	const detailSource = readFileSync(new URL('../src/pages/news/components/BattleReportDetail.tsx', import.meta.url), 'utf8')
	assert.match(detailSource, /traits: general\.traits/)
	const start = detailSource.indexOf('function sideTriggeredEffectText')
	const end = detailSource.indexOf('\n}\n\n// sidePassiveEffectText', start)
	assert.ok(start >= 0 && end > start)
	const effectBody = detailSource.slice(start, end)
	assert.match(effectBody, /detail\.traits/)
	assert.doesNotMatch(effectBody, /general\.traits/)
})

test('纯攻击加成特性明确限制为主动进攻并展示实际攻击变化', () => {
	const options = { faction: 'wei', units: { wei: { weiInfantry: { name: '魏步兵' } } } }
	for (const traitId of ['sizhandaodi', 'weizhen_xiaoyao', 'wusheng_pojun', 'wanren_nuhou', 'xiaobawang_tieqi', 'meizhoulang_junlue']) {
		const trait = getTraitMeta(traitId)
		assert.equal(trait.trigger, '主动进攻战斗前')
		assert.match(trait.description, /仅在主动进攻/)
	}
	assert.equal(formatTraitOutcomeDetail('attackModifiedUnits', { weiInfantry: 4 }, options), '实际攻击修正: 魏步兵 +4')
	assert.equal(formatTraitOutcomeDetail('attackModifiedUnits', { overlordRider: 50 }), '实际攻击修正: 霸王骑 +50')
})

test('战前属性特性只在实际影响战力的方向触发', () => {
	const delayed = getTraitMeta('mouding_houfa')
	assert.equal(delayed.trigger, '防守/增援战斗前')
	assert.match(delayed.description, /防守或作为援军/)
	assert.match(delayed.description, /防御提升 35%/)
	for (const traitId of ['meihuo_raozhen', 'huchi_chongzhen', 'baibu_chuanyang', 'qibing_raohou']) {
		const trait = getTraitMeta(traitId)
		assert.equal(trait.trigger, '主动进攻战斗前')
		assert.match(trait.description, /仅在主动进攻/)
	}
	assert.match(getTraitMeta('meihuo_raozhen').description, /防御降低 25%/)
	assert.match(getTraitMeta('huchi_chongzhen').description, /50% 概率/)
	assert.match(getTraitMeta('huchi_chongzhen').description, /防御降低 30%/)
	assert.match(getTraitMeta('baibu_chuanyang').description, /防御降低 20%/)
	assert.match(getTraitMeta('qibing_raohou').description, /降低敌军 20%/)
	assert.equal(formatTraitOutcomeDetail('attackModifiedUnits', { huWei: -1 }), '实际攻击修正: 虎卫 -1')
	assert.equal(formatTraitOutcomeDetail('infantryDefenseModifiedUnits', { huWei: -3 }), '实际步防修正: 虎卫 -3')
	assert.equal(formatTraitOutcomeDetail('cavalryDefenseModifiedUnits', { huWei: -3 }), '实际骑防修正: 虎卫 -3')
	assert.equal(formatTraitOutcomeDetail('attackReductionRate', 0.1), '设计攻击降低: 10%')
	assert.equal(formatTraitOutcomeDetail('enemyDefenseReductionRate', 0.35), '设计敌方防御降低: 35%')
})

test('司马懿守城双特性归属防守侧并分别展示真实伤亡和加防', () => {
	const detail = {
		sourceType: 'player_city', viewType: 'attack',
		traits: [
			{
				traitId: 'yibing_touxi', traitName: '疑兵偷袭', ownerSide: 'secondary', ownerRole: 'defender', generalId: 'simayi',
				detail: { effectRate: 0.35, preBattleAffected: { greedyWolf: 350 }, triggerChance: 1 },
			},
			{
				traitId: 'mouding_houfa', traitName: '谋定后发', ownerSide: 'secondary', ownerRole: 'defender', generalId: 'simayi',
				detail: { defenseBonusRate: 0.35, infantryDefenseModifiedUnits: { weiInfantry: 4 }, cavalryDefenseModifiedUnits: { weiInfantry: 3 }, triggerChance: 1 },
			},
		],
	}
	assert.equal(hasTraitEntries(detail), true)
	assert.deepEqual(detail.traits.map((trait) => resolveReportTraitDisplaySide(detail, trait)), ['secondary', 'secondary'])
	assert.deepEqual(detail.traits.map((trait) => getTraitMeta(trait.traitId).trigger), ['进攻/防守/增援战斗前', '防守/增援战斗前'])
	assert.equal(formatTraitOutcomeDetail('preBattleAffected', detail.traits[0].detail.preBattleAffected), '战前真实伤亡: 贪狼营 +350')
	assert.equal(formatTraitOutcomeDetail('defenseBonusRate', detail.traits[1].detail.defenseBonusRate), '设计防御加成: 35%')
	assert.equal(formatTraitOutcomeDetail('infantryDefenseModifiedUnits', detail.traits[1].detail.infantryDefenseModifiedUnits), '实际步防修正: weiInfantry +4')
	assert.equal(formatTraitOutcomeDetail('cavalryDefenseModifiedUnits', detail.traits[1].detail.cavalryDefenseModifiedUnits), '实际骑防修正: weiInfantry +3')
})

test('防守司马懿疑兵未命中时只展示谋定后发真实加防', () => {
	const detail = {
		sourceType: 'player_city', viewType: 'defense', battleType: 'plunder', ownerSide: 'defender',
		primarySide: {
			role: 'attacker', power: 10000,
			units: [{ unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 1000, dispatched: 1000, lost: 617, survived: 383 }],
		},
		secondarySide: {
			role: 'defender', power: 14000,
			generals: [{
				id: 'simayi', name: '司马懿', level: 1,
				traits: [{ traitId: 'yibing_touxi', name: '疑兵偷袭' }, { traitId: 'mouding_houfa', name: '谋定后发' }],
			}],
			units: [{ unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 1000, dispatched: 1000, lost: 382, survived: 618 }],
		},
		rewards: { generalExp: 617, resources: {} },
		traits: [{
			traitId: 'mouding_houfa', traitName: '谋定后发', ownerSide: 'secondary', ownerRole: 'defender', generalId: 'simayi',
			detail: { defenseBonusRate: 0.35, infantryDefenseModifiedUnits: { weiInfantry: 4 }, cavalryDefenseModifiedUnits: { weiInfantry: 3 }, triggerChance: 1 },
		}],
	}
	const options = { faction: 'shu', units: { shu: { shuInfantry: { name: '蜀步兵' } }, wei: { weiInfantry: { name: '魏步兵' } } } }
	assert.deepEqual(detail.secondarySide.generals[0].traits.map((trait) => trait.traitId), ['yibing_touxi', 'mouding_houfa'])
	assert.deepEqual(detail.traits.map((trait) => trait.traitId), ['mouding_houfa'])
	assert.equal(detail.traits.some((trait) => trait.traitId === 'yibing_touxi'), false)
	assert.deepEqual(detail.traits.map((trait) => resolveReportTraitDisplaySide(detail, trait)), ['secondary'])
	assert.deepEqual(detail.primarySide.units[0], { unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 1000, dispatched: 1000, lost: 617, survived: 383 })
	assert.deepEqual(detail.secondarySide.units[0], { unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 1000, dispatched: 1000, lost: 382, survived: 618 })
	assert.equal(formatTraitOutcomeDetail('defenseBonusRate', detail.traits[0].detail.defenseBonusRate, options), '设计防御加成: 35%')
	assert.equal(formatTraitOutcomeDetail('infantryDefenseModifiedUnits', detail.traits[0].detail.infantryDefenseModifiedUnits, options), '实际步防修正: 魏步兵 +4')
	assert.deepEqual(detail.rewards, { generalExp: 617, resources: {} })
})

test('司马懿黄巾守城按疑兵伤亡再谋定加防的顺序展示组合结果', () => {
	const options = { faction: 'wei', units: { wei: { weiInfantry: { name: '魏步兵' } } } }
	const detail = {
		sourceType: 'yellow_turban', viewType: 'defense',
		primarySide: {
			role: 'attacker', power: 6500,
			units: [{ unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 1000, dispatched: 1000, lost: 1000, survived: 0 }],
		},
		secondarySide: {
			role: 'defender', power: 14420,
			generals: [{ id: 'simayi', name: '司马懿', traits: [{ traitId: 'yibing_touxi' }, { traitId: 'mouding_houfa' }] }],
			units: [{ unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 1000, dispatched: 1000, lost: 312, survived: 688 }],
		},
		traits: [
			{
				traitId: 'yibing_touxi', traitName: '疑兵偷袭', ownerSide: 'secondary', ownerRole: 'defender', generalId: 'simayi',
				detail: { effectRate: 0.35, preBattleAffected: { weiInfantry: 350 }, triggerChance: 1 },
			},
			{
				traitId: 'mouding_houfa', traitName: '谋定后发', ownerSide: 'secondary', ownerRole: 'defender', generalId: 'simayi',
				detail: { defenseBonusRate: 0.35, infantryDefenseModifiedUnits: { weiInfantry: 4 }, cavalryDefenseModifiedUnits: { weiInfantry: 3 }, triggerChance: 1 },
			},
		],
	}
	assert.deepEqual(detail.traits.map((trait) => trait.traitId), ['yibing_touxi', 'mouding_houfa'])
	assert.deepEqual(detail.traits.map((trait) => resolveReportTraitDisplaySide(detail, trait)), ['secondary', 'secondary'])
	assert.equal(detail.primarySide.power, 6500)
	assert.equal(formatTraitOutcomeDetail('preBattleAffected', detail.traits[0].detail.preBattleAffected, options), '战前真实伤亡: 魏步兵 +350')
	assert.equal(formatTraitOutcomeDetail('infantryDefenseModifiedUnits', detail.traits[1].detail.infantryDefenseModifiedUnits, options), '实际步防修正: 魏步兵 +4')
})

test('关羽与张飞黄巾守城随机战前特性命中或未命中时保持准确战报', () => {
	const cases = [
		{
			generalId: 'guanyu', generalName: '关羽', defenderFaction: 'shu', defenderUnit: 'shuInfantry', defenderUnitName: '蜀步兵',
			attackerFaction: 'wei', attackerUnit: 'weiInfantry', attackerUnitName: '魏步兵', traitId: 'shuiyan_qijun', traitName: '水淹七军', bonusId: 'wusheng_pojun', bonusName: '武圣破军',
			defensePower: 1020, hitAttackPower: 650, hitDefenseLost: 52, hitAttackLost: 100, missDefenseLost: 97,
			detail: { effectRate: 0.35, maxAffectedRate: 0.35, preBattleAffected: { weiInfantry: 35 }, triggerChance: 1 },
		},
		{
			generalId: 'zhangfei', generalName: '张飞', defenderFaction: 'shu', defenderUnit: 'shuInfantry', defenderUnitName: '蜀步兵',
			attackerFaction: 'wei', attackerUnit: 'weiInfantry', attackerUnitName: '魏步兵', traitId: 'zhenhe_quanjun', traitName: '震慑全军', bonusId: 'wanren_nuhou', bonusName: '万人怒吼',
			defensePower: 1020, hitAttackPower: 500, hitDefenseLost: 36, hitAttackLost: 50, missDefenseLost: 97,
			detail: { effectRate: 0.5, maxAffectedRate: 0.5, suppressedUnits: { weiInfantry: 50 }, triggerChance: 1 },
		},
	] as const
	const options = { units: { shu: { shuInfantry: { name: '蜀步兵' } }, wei: { weiInfantry: { name: '魏步兵' } } } }

	for (const current of cases) {
		const buildReport = (triggered: boolean) => {
			const attackerLost = triggered ? current.hitAttackLost : 100
			const defenderLost = triggered ? current.hitDefenseLost : current.missDefenseLost
			return {
				id: `${current.generalId}-yellow-${triggered ? 'hit' : 'miss'}`, playerId: `player_${current.generalId}`, ownerPlayerId: `player_${current.generalId}`,
				viewType: 'defense', sourceType: 'yellow_turban', battleType: 'yellow_turban', type: 'defense', result: 'defender_victory', rewards: {}, read: true, createdAt: '2026-07-20T23:30:00Z',
				detail: {
					id: `${current.generalId}-yellow-${triggered ? 'hit' : 'miss'}`, sourceType: 'yellow_turban', viewType: 'defense', battleType: 'yellow_turban',
					result: 'defender_victory', winnerSide: 'defender', ownerSide: 'defender',
					primarySide: {
						role: 'attacker', faction: current.attackerFaction, power: triggered ? current.hitAttackPower : 1000,
						units: [{ unitType: current.attackerUnit, unitName: current.attackerUnitName, amountBefore: 100, dispatched: 100, lost: attackerLost, survived: 100 - attackerLost }],
					},
					secondarySide: {
						role: 'defender', faction: current.defenderFaction, power: current.defensePower,
						generals: [{
							id: current.generalId, name: current.generalName, level: 1,
							traits: [{ traitId: current.traitId, name: current.traitName }, { traitId: current.bonusId, name: current.bonusName, allowedSides: ['attacker'] }],
						}],
						units: [{ unitType: current.defenderUnit, unitName: current.defenderUnitName, amountBefore: 100, dispatched: 100, lost: defenderLost, survived: 100 - defenderLost }],
					},
					rewards: { generalExp: attackerLost, resources: {} },
					traits: triggered ? [{
						traitId: current.traitId, traitName: current.traitName, ownerSide: 'secondary', ownerRole: 'defender', ownerPlayerId: `player_${current.generalId}`, generalId: current.generalId,
						detail: current.detail,
					}] : [],
				},
			}
		}
		const hit = normalizeBattleReportDetail(buildReport(true))
		const miss = normalizeBattleReportDetail(buildReport(false))

		assert.deepEqual(hit.primarySide.units[0], { unitType: current.attackerUnit, unitName: current.attackerUnitName, amountBefore: 100, dispatched: 100, lost: current.hitAttackLost, survived: 100 - current.hitAttackLost })
		assert.deepEqual(miss.primarySide.units[0], { unitType: current.attackerUnit, unitName: current.attackerUnitName, amountBefore: 100, dispatched: 100, lost: 100, survived: 0 })
		assert.deepEqual(hit.secondarySide?.units[0], { unitType: current.defenderUnit, unitName: current.defenderUnitName, amountBefore: 100, dispatched: 100, lost: current.hitDefenseLost, survived: 100 - current.hitDefenseLost })
		assert.deepEqual(miss.secondarySide?.units[0], { unitType: current.defenderUnit, unitName: current.defenderUnitName, amountBefore: 100, dispatched: 100, lost: current.missDefenseLost, survived: 100 - current.missDefenseLost })
		assert.equal(hit.primarySide.power, current.hitAttackPower)
		assert.equal(miss.primarySide.power, 1000)
		assert.equal(hit.secondarySide?.power, current.defensePower)
		assert.equal(miss.secondarySide?.power, current.defensePower)
		assert.equal(hit.rewards.generalExp, current.hitAttackLost)
		assert.equal(miss.rewards.generalExp, 100)
		for (const detail of [hit, miss]) {
			assert.deepEqual(detail.secondarySide?.generals?.[0]?.traits?.map((trait) => trait.traitId), [current.traitId, current.bonusId])
		}
		assert.deepEqual(hit.traits?.map((trait) => trait.traitId), [current.traitId])
		assert.equal(resolveReportTraitDisplaySide(hit, hit.traits?.[0]), 'secondary')
		assert.equal(hasTraitEntries(miss), false)
		assert.equal([...(hit.traits ?? []), ...(miss.traits ?? [])].some((trait) => trait.traitId === current.bonusId), false)
		if (current.traitId === 'shuiyan_qijun') {
			assert.equal(formatTraitOutcomeDetail('preBattleAffected', hit.traits?.[0].detail.preBattleAffected, options), '战前真实伤亡: 魏步兵 +35')
		} else {
			assert.equal(formatTraitOutcomeDetail('suppressedUnits', hit.traits?.[0].detail.suppressedUnits, options), '本场压制兵力: 魏步兵 +50')
		}
	}
})

test('陆逊黄巾守城火烧命中或未命中时只展示实际生效的战后伤害', () => {
	const buildReport = (triggered: boolean) => {
		const attackerLost = triggered ? 200 : 97
		const traitId = triggered ? 'huoshao_lianying' : 'lianying_zengshang'
		const traitName = triggered ? '火烧联营' : '连营增伤'
		return {
			id: `luxun-yellow-${triggered ? 'hit' : 'miss'}`, playerId: 'player_luxun', ownerPlayerId: 'player_luxun',
			viewType: 'defense', sourceType: 'yellow_turban', battleType: 'yellow_turban', type: 'defense', result: 'attacker_victory', rewards: {}, read: true, createdAt: '2026-07-21T00:10:00Z',
			detail: {
				id: `luxun-yellow-${triggered ? 'hit' : 'miss'}`, sourceType: 'yellow_turban', viewType: 'defense', battleType: 'yellow_turban',
				result: 'attacker_victory', winnerSide: 'attacker', ownerSide: 'defender',
				primarySide: {
					role: 'attacker', faction: 'wei', power: 2000,
					units: [{ unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 200, dispatched: 200, lost: attackerLost, survived: 200 - attackerLost }],
				},
				secondarySide: {
					role: 'defender', faction: 'wu', power: 1025,
					generals: [{
						id: 'luxun', name: '陆逊', level: 1,
						traits: [{ traitId: 'huoshao_lianying', name: '火烧联营' }, { traitId: 'lianying_zengshang', name: '连营增伤' }],
					}],
					units: [{ unitType: 'wuInfantry', unitName: '吴步兵', amountBefore: 100, dispatched: 100, lost: 100, survived: 0 }],
				},
				rewards: { generalExp: attackerLost, resources: {} },
				traits: [{
					traitId, traitName, ownerSide: 'secondary', ownerRole: 'defender', ownerPlayerId: 'player_luxun', generalId: 'luxun',
					detail: triggered
						? { effectRate: 1, maxAffectedRate: 1, targetExtraLosses: { weiInfantry: 123 }, triggerChance: 1 }
						: { effectRate: 0.1, maxAffectedRate: 0.1, targetExtraLosses: { weiInfantry: 20 }, triggerChance: 1 },
				}],
			},
		}
	}
	const hit = normalizeBattleReportDetail(buildReport(true))
	const miss = normalizeBattleReportDetail(buildReport(false))
	const options = { faction: 'wei', units: { wei: { weiInfantry: { name: '魏步兵' } } } }

	assert.deepEqual(hit.primarySide.units[0], { unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 200, dispatched: 200, lost: 200, survived: 0 })
	assert.deepEqual(miss.primarySide.units[0], { unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 200, dispatched: 200, lost: 97, survived: 103 })
	for (const detail of [hit, miss]) {
		assert.equal(detail.primarySide.power, 2000)
		assert.equal(detail.secondarySide?.power, 1025)
		assert.deepEqual(detail.secondarySide?.units[0], { unitType: 'wuInfantry', unitName: '吴步兵', amountBefore: 100, dispatched: 100, lost: 100, survived: 0 })
		assert.deepEqual(detail.secondarySide?.generals?.[0]?.traits?.map((trait) => trait.traitId), ['huoshao_lianying', 'lianying_zengshang'])
		assert.equal(resolveReportTraitDisplaySide(detail, detail.traits?.[0]), 'secondary')
	}
	assert.equal(hit.rewards.generalExp, 200)
	assert.equal(miss.rewards.generalExp, 97)
	assert.deepEqual(hit.traits?.map((trait) => trait.traitId), ['huoshao_lianying'])
	assert.deepEqual(miss.traits?.map((trait) => trait.traitId), ['lianying_zengshang'])
	assert.equal(formatTraitOutcomeDetail('targetExtraLosses', hit.traits?.[0].detail.targetExtraLosses, options), '目标兵种追加损失: 魏步兵 +123')
	assert.equal(formatTraitOutcomeDetail('targetExtraLosses', miss.traits?.[0].detail.targetExtraLosses, options), '目标兵种追加损失: 魏步兵 +20')
	assert.equal(hit.traits?.some((trait) => trait.traitId === 'lianying_zengshang'), false)
	assert.equal(miss.traits?.some((trait) => trait.traitId === 'huoshao_lianying'), false)
})

test('黄盖黄巾守城苦肉命中记录零压制且未命中不影响反击', () => {
	const buildReport = (triggered: boolean) => ({
		id: `huanggai-yellow-${triggered ? 'hit' : 'miss'}`, playerId: 'player_huanggai', ownerPlayerId: 'player_huanggai',
		viewType: 'defense', sourceType: 'yellow_turban', battleType: 'yellow_turban', type: 'defense', result: 'attacker_victory', rewards: {}, read: true, createdAt: '2026-07-21T00:30:00Z',
		detail: {
			id: `huanggai-yellow-${triggered ? 'hit' : 'miss'}`, sourceType: 'yellow_turban', viewType: 'defense', battleType: 'yellow_turban',
			result: 'attacker_victory', winnerSide: 'attacker', ownerSide: 'defender',
			primarySide: {
				role: 'attacker', faction: 'wei', power: 2000,
				units: [{ unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 200, dispatched: 200, lost: 97, survived: 103 }],
			},
			secondarySide: {
				role: 'defender', faction: 'wu', power: 1025,
				generals: [{
					id: 'huanggai', name: '黄盖', level: 1,
					traits: [{ traitId: 'kurouji', name: '苦肉计' }, { traitId: 'kurou_fanji', name: '苦肉反击' }],
				}],
				units: [{ unitType: 'wuInfantry', unitName: '吴步兵', amountBefore: 100, dispatched: 100, lost: 100, survived: 0 }],
			},
			rewards: { generalExp: 97, resources: {} },
			traits: [
				...(triggered ? [{
					traitId: 'kurouji', traitName: '苦肉计', ownerSide: 'secondary', ownerRole: 'defender', ownerPlayerId: 'player_huanggai', generalId: 'huanggai',
					detail: { disableTraitCount: 1, disabledTraitCount: 0, triggerChance: 1 },
				}] : []),
				{
					traitId: 'kurou_fanji', traitName: '苦肉反击', ownerSide: 'secondary', ownerRole: 'defender', ownerPlayerId: 'player_huanggai', generalId: 'huanggai',
					detail: { effectRate: 0.1, extraLosses: { weiInfantry: 20 }, triggerChance: 1 },
				},
			],
		},
	})
	const hit = normalizeBattleReportDetail(buildReport(true))
	const miss = normalizeBattleReportDetail(buildReport(false))
	const options = { faction: 'wei', units: { wei: { weiInfantry: { name: '魏步兵' } } } }

	for (const detail of [hit, miss]) {
		assert.equal(detail.primarySide.power, 2000)
		assert.equal(detail.secondarySide?.power, 1025)
		assert.deepEqual(detail.primarySide.units[0], { unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 200, dispatched: 200, lost: 97, survived: 103 })
		assert.deepEqual(detail.secondarySide?.units[0], { unitType: 'wuInfantry', unitName: '吴步兵', amountBefore: 100, dispatched: 100, lost: 100, survived: 0 })
		assert.deepEqual(detail.secondarySide?.generals?.[0]?.traits?.map((trait) => trait.traitId), ['kurouji', 'kurou_fanji'])
		assert.equal(detail.rewards.generalExp, 97)
		assert.deepEqual(detail.traits?.map((trait) => resolveReportTraitDisplaySide(detail, trait)), detail.traits?.map(() => 'secondary'))
	}
	assert.deepEqual(hit.traits?.map((trait) => trait.traitId), ['kurouji', 'kurou_fanji'])
	assert.deepEqual(miss.traits?.map((trait) => trait.traitId), ['kurou_fanji'])
	assert.equal(formatTraitOutcomeDetail('disableTraitCount', hit.traits?.[0].detail.disableTraitCount, options), '设计压制特性数: 1')
	assert.equal(formatTraitOutcomeDetail('disabledTraitCount', hit.traits?.[0].detail.disabledTraitCount, options), '实际压制特性数: 0')
	assert.equal(formatTraitOutcomeDetail('extraLosses', hit.traits?.[1].detail.extraLosses, options), '追加损失: 魏步兵 +20')
	assert.equal(formatTraitOutcomeDetail('extraLosses', miss.traits?.[0].detail.extraLosses, options), '追加损失: 魏步兵 +20')
	assert.equal(miss.traits?.some((trait) => trait.traitId === 'kurouji'), false)
})

test('马超黄巾守城西凉命中或未命中时保持骑兵损失和被动快照准确', () => {
	const buildReport = (triggered: boolean) => {
		const attackerLost = triggered ? 58 : 34
		const generalExp = triggered ? 116 : 68
		return {
			id: `machao-yellow-${triggered ? 'hit' : 'miss'}`, playerId: 'player_machao', ownerPlayerId: 'player_machao',
			viewType: 'defense', sourceType: 'yellow_turban', battleType: 'yellow_turban', type: 'defense', result: 'attacker_victory', rewards: {}, read: true, createdAt: '2026-07-21T01:00:00Z',
			detail: {
				id: `machao-yellow-${triggered ? 'hit' : 'miss'}`, sourceType: 'yellow_turban', viewType: 'defense', battleType: 'yellow_turban',
				result: 'attacker_victory', winnerSide: 'attacker', ownerSide: 'defender',
				primarySide: {
					role: 'attacker', faction: 'wei', power: 2800,
					units: [{ unitType: 'weiCavalry', unitName: '魏骑兵', amountBefore: 200, dispatched: 200, lost: attackerLost, survived: 200 - attackerLost }],
				},
				secondarySide: {
					role: 'defender', faction: 'shu', power: 816,
					generals: [{
						id: 'machao', name: '马超', level: 1,
						stats: { force: 0, intelligence: 0, command: 0, politics: 0 }, effectiveStats: { force: 20, intelligence: 0, command: 0, politics: 0 },
						buffs: { attackBonus: 0.4 }, traits: [{ traitId: 'xiliang_tuji', name: '西凉突击' }, { traitId: 'tianshen_xiafan', name: '天神下凡' }],
					}],
					units: [{ unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 100, dispatched: 100, lost: 100, survived: 0 }],
				},
				rewards: { generalExp, resources: {} },
				traits: triggered ? [{
					traitId: 'xiliang_tuji', traitName: '西凉突击', ownerSide: 'secondary', ownerRole: 'defender', ownerPlayerId: 'player_machao', generalId: 'machao',
					detail: { effectRate: 0.12, targetExtraLosses: { weiCavalry: 24 }, triggerChance: 1 },
				}] : [],
			},
		}
	}
	const hit = normalizeBattleReportDetail(buildReport(true))
	const miss = normalizeBattleReportDetail(buildReport(false))
	const options = { faction: 'wei', units: { wei: { weiCavalry: { name: '魏骑兵' } } } }

	assert.deepEqual(hit.primarySide.units[0], { unitType: 'weiCavalry', unitName: '魏骑兵', amountBefore: 200, dispatched: 200, lost: 58, survived: 142 })
	assert.deepEqual(miss.primarySide.units[0], { unitType: 'weiCavalry', unitName: '魏骑兵', amountBefore: 200, dispatched: 200, lost: 34, survived: 166 })
	for (const detail of [hit, miss]) {
		assert.equal(detail.primarySide.power, 2800)
		assert.equal(detail.secondarySide?.power, 816)
		assert.deepEqual(detail.secondarySide?.units[0], { unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 100, dispatched: 100, lost: 100, survived: 0 })
		assert.deepEqual(detail.secondarySide?.generals?.[0]?.traits?.map((trait) => trait.traitId), ['xiliang_tuji', 'tianshen_xiafan'])
		assert.equal(detail.secondarySide?.generals?.[0]?.effectiveStats?.force, 20)
		assert.equal(detail.secondarySide?.generals?.[0]?.buffs?.attackBonus, 0.4)
	}
	assert.equal(hit.rewards.generalExp, 116)
	assert.equal(miss.rewards.generalExp, 68)
	assert.deepEqual(hit.traits?.map((trait) => trait.traitId), ['xiliang_tuji'])
	assert.equal(resolveReportTraitDisplaySide(hit, hit.traits?.[0]), 'secondary')
	assert.equal(formatTraitOutcomeDetail('targetExtraLosses', hit.traits?.[0].detail.targetExtraLosses, options), '目标兵种追加损失: 魏骑兵 +24')
	assert.equal(hasTraitEntries(miss), false)
	assert.equal([...(hit.traits ?? []), ...(miss.traits ?? [])].some((trait) => trait.traitId === 'tianshen_xiafan'), false)
})

test('赵云与孙权黄巾守城合法未命中时保持基础兵损和空时间线', () => {
	const cases = [
		{
			generalId: 'zhaoyun', generalName: '赵云', traitIds: ['longdan_jiuyuan', 'qijin_qichu'],
			attackerAmount: 200, attackerPower: 2000, attackerLost: 76, attackerSurvived: 124,
			defenderFaction: 'shu', defenderUnit: 'greedyWolf', defenderUnitName: '贪狼营', defenderPower: 1020, defenderLost: 100, defenderSurvived: 0, generalExp: 76,
		},
		{
			generalId: 'sunquan', generalName: '孙权', traitIds: ['jiangdong_haoling', 'jiangdong_gushou'],
			attackerAmount: 100, attackerPower: 1000, attackerLost: 100, attackerSurvived: 0,
			defenderFaction: 'wu', defenderUnit: 'wuInfantry', defenderUnitName: '吴步兵', defenderPower: 1025, defenderLost: 96, defenderSurvived: 4, generalExp: 100,
		},
	] as const

	for (const current of cases) {
		const detail = normalizeBattleReportDetail({
			id: `${current.generalId}-yellow-miss`, playerId: `player_${current.generalId}`, ownerPlayerId: `player_${current.generalId}`,
			viewType: 'defense', sourceType: 'yellow_turban', battleType: 'yellow_turban', type: 'defense',
			result: current.generalId === 'zhaoyun' ? 'attacker_victory' : 'defender_victory', rewards: {}, read: true, createdAt: '2026-07-21T01:05:00Z',
			detail: {
				id: `${current.generalId}-yellow-miss`, sourceType: 'yellow_turban', viewType: 'defense', battleType: 'yellow_turban',
				result: current.generalId === 'zhaoyun' ? 'attacker_victory' : 'defender_victory', winnerSide: current.generalId === 'zhaoyun' ? 'attacker' : 'defender', ownerSide: 'defender',
				primarySide: {
					role: 'attacker', faction: 'wei', power: current.attackerPower,
					units: [{ unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: current.attackerAmount, dispatched: current.attackerAmount, lost: current.attackerLost, survived: current.attackerSurvived }],
				},
				secondarySide: {
					role: 'defender', faction: current.defenderFaction, power: current.defenderPower,
					generals: [{ id: current.generalId, name: current.generalName, level: 1, traits: current.traitIds.map((traitId) => ({ traitId, name: getTraitMeta(traitId).name })) }],
					units: [{ unitType: current.defenderUnit, unitName: current.defenderUnitName, amountBefore: 100, dispatched: 100, lost: current.defenderLost, survived: current.defenderSurvived }],
				},
				rewards: { generalExp: current.generalExp, resources: {} }, traits: [],
			},
		})
		assert.equal(detail.primarySide.power, current.attackerPower)
		assert.equal(detail.secondarySide?.power, current.defenderPower)
		assert.deepEqual(detail.primarySide.units[0], { unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: current.attackerAmount, dispatched: current.attackerAmount, lost: current.attackerLost, survived: current.attackerSurvived })
		assert.deepEqual(detail.secondarySide?.units[0], { unitType: current.defenderUnit, unitName: current.defenderUnitName, amountBefore: 100, dispatched: 100, lost: current.defenderLost, survived: current.defenderSurvived })
		assert.deepEqual(detail.secondarySide?.generals?.[0]?.traits?.map((trait) => trait.traitId), [...current.traitIds])
		assert.equal(detail.rewards.generalExp, current.generalExp)
		assert.equal(hasTraitEntries(detail), false)
	}
})

test('司马懿黄巾与 NPC 疑兵命中或未命中时严格隔离谋定方向', () => {
	const cases = [
		{
			id: 'simayi-yellow-miss', sourceType: 'yellow_turban', viewType: 'defense', result: 'defender_victory', winnerSide: 'defender', ownerSide: 'defender',
			primaryPower: 10000, primaryAmount: 1000, primaryLost: 1000, primarySurvived: 0,
			secondaryPower: 14420, secondaryAmount: 1000, secondaryLost: 594, secondarySurvived: 406, generalExp: 1000,
			generalSide: 'secondary', traits: [{
				traitId: 'mouding_houfa', traitName: '谋定后发', ownerSide: 'secondary', ownerRole: 'defender', ownerPlayerId: 'player_simayi', generalId: 'simayi',
				detail: { defenseBonusRate: 0.35, infantryDefenseModifiedUnits: { weiInfantry: 4 }, cavalryDefenseModifiedUnits: { weiInfantry: 3 }, triggerChance: 1 },
			}],
		},
		{
			id: 'simayi-npc-hit', sourceType: 'npc_city', viewType: 'attack', result: 'attacker_victory', winnerSide: 'attacker', ownerSide: 'attacker',
			primaryPower: 2000, primaryAmount: 200, primaryLost: 40, primarySurvived: 160,
			secondaryPower: 650, secondaryAmount: 100, secondaryLost: 100, secondarySurvived: 0, generalExp: 100,
			generalSide: 'primary', traits: [{
				traitId: 'yibing_touxi', traitName: '疑兵偷袭', ownerSide: 'primary', ownerRole: 'attacker', ownerPlayerId: 'player_simayi', generalId: 'simayi',
				detail: { effectRate: 0.35, preBattleAffected: { weiInfantry: 35 }, triggerChance: 1 },
			}],
		},
		{
			id: 'simayi-npc-miss', sourceType: 'npc_city', viewType: 'attack', result: 'attacker_victory', winnerSide: 'attacker', ownerSide: 'attacker',
			primaryPower: 2000, primaryAmount: 200, primaryLost: 74, primarySurvived: 126,
			secondaryPower: 1000, secondaryAmount: 100, secondaryLost: 100, secondarySurvived: 0, generalExp: 100,
			generalSide: 'primary', traits: [],
		},
	] as const

	for (const current of cases) {
		const general = { id: 'simayi', name: '司马懿', level: 1, traits: [{ traitId: 'yibing_touxi', name: '疑兵偷袭' }, { traitId: 'mouding_houfa', name: '谋定后发', allowedSides: ['defender', 'reinforcement'] }] }
		const detail = normalizeBattleReportDetail({
			id: current.id, playerId: 'player_simayi', ownerPlayerId: 'player_simayi', viewType: current.viewType, sourceType: current.sourceType,
			battleType: current.sourceType === 'yellow_turban' ? 'yellow_turban' : 'attack', type: current.viewType, result: current.result, rewards: {}, read: true, createdAt: '2026-07-21T02:00:00Z',
			detail: {
				id: current.id, sourceType: current.sourceType, viewType: current.viewType, battleType: current.sourceType === 'yellow_turban' ? 'yellow_turban' : 'attack',
				result: current.result, winnerSide: current.winnerSide, ownerSide: current.ownerSide,
				primarySide: {
					role: 'attacker', faction: 'wei', power: current.primaryPower, generals: current.generalSide === 'primary' ? [general] : [],
					units: [{ unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: current.primaryAmount, dispatched: current.primaryAmount, lost: current.primaryLost, survived: current.primarySurvived }],
				},
				secondarySide: {
					role: 'defender', faction: 'wei', power: current.secondaryPower, generals: current.generalSide === 'secondary' ? [general] : [],
					units: [{ unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: current.secondaryAmount, dispatched: current.secondaryAmount, lost: current.secondaryLost, survived: current.secondarySurvived }],
				},
				rewards: { generalExp: current.generalExp, resources: {} }, traits: [...current.traits],
			},
		})
		assert.deepEqual(detail.primarySide.units[0], { unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: current.primaryAmount, dispatched: current.primaryAmount, lost: current.primaryLost, survived: current.primarySurvived })
		assert.deepEqual(detail.secondarySide?.units[0], { unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: current.secondaryAmount, dispatched: current.secondaryAmount, lost: current.secondaryLost, survived: current.secondarySurvived })
		assert.deepEqual((current.generalSide === 'primary' ? detail.primarySide : detail.secondarySide)?.generals?.[0]?.traits?.map((trait) => trait.traitId), ['yibing_touxi', 'mouding_houfa'])
		assert.deepEqual(detail.traits?.map((trait) => trait.traitId), current.traits.map((trait) => trait.traitId))
		assert.equal(detail.traits?.some((trait) => trait.traitId === 'mouding_houfa'), current.sourceType === 'yellow_turban')
		assert.equal(detail.traits?.some((trait) => trait.traitId === 'yibing_touxi'), current.id === 'simayi-npc-hit')
		assert.equal(detail.rewards.generalExp, current.generalExp)
	}
})

test('典韦黄巾防守与 NPC 进攻分别展示护主血战和死战到底的实际结果', () => {
	const general = { id: 'dianwei', name: '典韦', level: 1, traits: [{ traitId: 'huzhu_xuezhan', name: '护主血战', allowedSides: ['defender', 'reinforcement'] }, { traitId: 'sizhandaodi', name: '死战到底', allowedSides: ['attacker'] }] }
	const yellow = normalizeBattleReportDetail({
		id: 'dianwei-yellow-defense', playerId: 'player_dianwei', ownerPlayerId: 'player_dianwei', viewType: 'defense', sourceType: 'yellow_turban', battleType: 'yellow_turban', type: 'defense', result: 'attacker_victory', rewards: {}, read: true, createdAt: '2026-07-23T02:05:00Z',
		detail: {
			id: 'dianwei-yellow-defense', sourceType: 'yellow_turban', viewType: 'defense', battleType: 'yellow_turban', result: 'attacker_victory', winnerSide: 'attacker', ownerSide: 'defender',
			primarySide: { role: 'attacker', faction: 'wei', power: 2000, generals: [], units: [{ unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 200, dispatched: 200, lost: 200, survived: 0 }] },
			secondarySide: { role: 'defender', faction: 'wei', power: 3399, generals: [general], units: [{ unitType: 'jinWeiSoldier', unitName: '禁卫甲士', amountBefore: 100, dispatched: 100, lost: 47, survived: 53 }] },
			rewards: { generalExp: 200, resources: {} }, traits: [{ traitId: 'huzhu_xuezhan', traitName: '护主血战', ownerSide: 'secondary', ownerRole: 'defender', ownerPlayerId: 'player_dianwei', generalId: 'dianwei', detail: { generalDefenseFlat: 20, infantryDefenseModifiedUnits: { jinWeiSoldier: 20 }, cavalryDefenseModifiedUnits: { jinWeiSoldier: 20 }, triggerChance: 1 } }],
		},
	})
	assert.deepEqual(yellow.secondarySide?.units[0], { unitType: 'jinWeiSoldier', unitName: '禁卫甲士', amountBefore: 100, dispatched: 100, lost: 47, survived: 53 })
	assert.deepEqual(yellow.secondarySide?.generals?.[0]?.traits?.map((trait) => trait.traitId), ['huzhu_xuezhan', 'sizhandaodi'])
	assert.deepEqual(yellow.traits?.map((trait) => trait.traitId), ['huzhu_xuezhan'])

	for (const hit of [true, false]) {
		const npc = normalizeBattleReportDetail({
			id: `dianwei-npc-${hit ? 'hit' : 'miss'}`, playerId: 'player_dianwei', ownerPlayerId: 'player_dianwei', viewType: 'attack', sourceType: 'npc_city', battleType: 'attack', type: 'attack', result: 'defender_victory', rewards: {}, read: true, createdAt: '2026-07-23T02:06:00Z',
			detail: {
				id: `dianwei-npc-${hit ? 'hit' : 'miss'}`, sourceType: 'npc_city', viewType: 'attack', battleType: 'attack', result: 'defender_victory', winnerSide: 'defender', ownerSide: 'attacker',
				primarySide: { role: 'attacker', faction: 'wei', power: hit ? 1400 : 1000, generals: [general], units: [{ unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 100, dispatched: 100, lost: 100, survived: 0 }] },
				secondarySide: { role: 'defender', faction: 'wei', power: 2000, generals: [], units: [{ unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 200, dispatched: 200, lost: hit ? 120 : 74, survived: hit ? 80 : 126 }] },
				rewards: { generalExp: hit ? 120 : 74, resources: {} }, traits: hit ? [{ traitId: 'sizhandaodi', traitName: '死战到底', ownerSide: 'primary', ownerRole: 'attacker', ownerPlayerId: 'player_dianwei', generalId: 'dianwei', detail: { attackBonusRate: 0.35, attackModifiedUnits: { weiInfantry: 4 }, triggerChance: 0.6 } }] : [],
			},
		})
		assert.equal(npc.primarySide.power, hit ? 1400 : 1000)
		assert.equal(npc.secondarySide?.units[0]?.lost, hit ? 120 : 74)
		assert.deepEqual(npc.traits?.map((trait) => trait.traitId), hit ? ['sizhandaodi'] : [])
	}
})

test('关羽张辽张飞 NPC 战前随机未命中时保留各自独立战斗加成', () => {
	const cases = [
		{
			generalId: 'guanyu', generalName: '关羽', attackerUnit: 'weiInfantry', attackerUnitName: '魏步兵', specialId: 'shuiyan_qijun', specialName: '水淹七军', bonusId: 'wusheng_pojun', bonusName: '武圣破军',
			hitPower: [2400, 650], missPower: [2400, 1000], hitLosses: [31, 100], missLosses: [57, 100], hitExp: 100, missExp: 100,
			hitSpecialDetail: { effectRate: 0.35, maxAffectedRate: 0.35, preBattleAffected: { weiInfantry: 35 }, triggerChance: 1 },
			hitBonusDetail: { attackBonusRate: 0.2, attackModifiedUnits: { weiInfantry: 2 }, triggerChance: 1 },
			missBonusDetail: { attackBonusRate: 0.2, attackModifiedUnits: { weiInfantry: 2 }, triggerChance: 1 },
		},
		{
			generalId: 'zhangliao', generalName: '张辽', attackerUnit: 'weiCavalry', attackerUnitName: '魏骑兵', specialId: 'weizhen_zhenhe', specialName: '震慑全军', bonusId: 'weizhen_xiaoyao', bonusName: '威震逍遥',
			hitPower: [3800, 600], missPower: [3800, 800], hitLosses: [14, 75], missLosses: [21, 100], hitExp: 75, missExp: 100,
			hitSpecialDetail: { effectRate: 0.25, suppressedUnits: { weiInfantry: 25 }, fledUnits: { weiInfantry: 25 }, returnedUnits: { weiInfantry: 25 }, triggerChance: 1 },
			hitBonusDetail: { attackBonusRate: 0.35, attackModifiedUnits: { weiCavalry: 5 }, triggerChance: 1 },
			missBonusDetail: { attackBonusRate: 0.35, attackModifiedUnits: { weiCavalry: 5 }, triggerChance: 1 },
		},
		{
			generalId: 'zhangfei', generalName: '张飞', attackerUnit: 'weiInfantry', attackerUnitName: '魏步兵', specialId: 'zhenhe_quanjun', specialName: '震慑全军', bonusId: 'wanren_nuhou', bonusName: '万人怒吼',
			hitPower: [2400, 500], missPower: [2400, 1000], hitLosses: [21, 50], missLosses: [57, 100], hitExp: 50, missExp: 100,
			hitSpecialDetail: { effectRate: 0.5, maxAffectedRate: 0.5, suppressedUnits: { weiInfantry: 50 }, triggerChance: 1 },
			hitBonusDetail: { attackBonusRate: 0.2, attackModifiedUnits: { weiInfantry: 2 }, triggerChance: 1 },
			missBonusDetail: { attackBonusRate: 0.2, attackModifiedUnits: { weiInfantry: 2 }, triggerChance: 1 },
		},
	] as const

	for (const current of cases) {
		for (const triggered of [true, false]) {
			const powers = triggered ? current.hitPower : current.missPower
			const losses = triggered ? current.hitLosses : current.missLosses
			const generalExp = triggered ? current.hitExp : current.missExp
			const detail = normalizeBattleReportDetail({
				id: `${current.generalId}-npc-${triggered ? 'hit' : 'miss'}`, playerId: `player_${current.generalId}`, ownerPlayerId: `player_${current.generalId}`,
				viewType: 'attack', sourceType: 'npc_city', battleType: 'attack', type: 'attack', result: 'attacker_victory', rewards: {}, read: true, createdAt: '2026-07-21T02:10:00Z',
				detail: {
					id: `${current.generalId}-npc-${triggered ? 'hit' : 'miss'}`, sourceType: 'npc_city', viewType: 'attack', battleType: 'attack', result: 'attacker_victory', winnerSide: 'attacker', ownerSide: 'attacker',
					primarySide: {
						role: 'attacker', faction: 'wei', power: powers[0],
						generals: [{ id: current.generalId, name: current.generalName, level: 1, traits: [{ traitId: current.specialId, name: current.specialName }, { traitId: current.bonusId, name: current.bonusName }] }],
						units: [{ unitType: current.attackerUnit, unitName: current.attackerUnitName, amountBefore: 200, dispatched: 200, lost: losses[0], survived: 200 - losses[0] }],
					},
					secondarySide: { role: 'defender', faction: 'wei', power: powers[1], units: [{ unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 100, dispatched: 100, lost: losses[1], survived: 100 - losses[1] }] },
					rewards: { generalExp, resources: {} },
					traits: [
						...(triggered ? [{ traitId: current.specialId, traitName: current.specialName, ownerSide: 'primary', ownerRole: 'attacker', ownerPlayerId: `player_${current.generalId}`, generalId: current.generalId, detail: current.hitSpecialDetail }] : []),
						{ traitId: current.bonusId, traitName: current.bonusName, ownerSide: 'primary', ownerRole: 'attacker', ownerPlayerId: `player_${current.generalId}`, generalId: current.generalId, detail: triggered ? current.hitBonusDetail : current.missBonusDetail },
					],
				},
			})
			assert.equal(detail.primarySide.power, powers[0])
			assert.equal(detail.secondarySide?.power, powers[1])
			assert.deepEqual(detail.primarySide.units[0], { unitType: current.attackerUnit, unitName: current.attackerUnitName, amountBefore: 200, dispatched: 200, lost: losses[0], survived: 200 - losses[0] })
			assert.deepEqual(detail.secondarySide?.units[0], { unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 100, dispatched: 100, lost: losses[1], survived: 100 - losses[1] })
			assert.deepEqual(detail.primarySide.generals?.[0]?.traits?.map((trait) => trait.traitId), [current.specialId, current.bonusId])
			assert.deepEqual(detail.traits?.map((trait) => trait.traitId), triggered ? [current.specialId, current.bonusId] : [current.bonusId])
			assert.equal(detail.traits?.some((trait) => trait.traitId === current.specialId), triggered)
			assert.equal(detail.rewards.generalExp, generalExp)
		}
	}
})

test('马超黄忠陆逊黄盖孙策 NPC 随机命中或未命中时保持后续效果和精确兵损', () => {
	const cases = [
		{
			generalId: 'machao', generalName: '马超', traitIds: ['xiliang_tuji', 'tianshen_xiafan'], playerUnit: 'weiInfantry', playerUnitName: '魏步兵', playerAmount: 1000, npcUnit: 'weiCavalry', npcUnitName: '魏骑兵', npcAmount: 1000,
			hitPower: [14000, 10000], missPower: [14000, 10000], hitLosses: [382, 737], missLosses: [382, 617],
			hitTraits: [{ traitId: 'xiliang_tuji', traitName: '西凉突击', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'machao', detail: { effectRate: 0.12, targetExtraLosses: { weiCavalry: 120 }, triggerChance: 1 } }], missTraits: [],
			snapshot: { stats: { force: 0 }, effectiveStats: { force: 20 }, buffs: { attackBonus: 0.4 } },
		},
		{
			generalId: 'huangzhong', generalName: '黄忠', traitIds: ['baibu_chuanyang', 'laodang_yizhuang'], playerUnit: 'weiInfantry', playerUnitName: '魏步兵', playerAmount: 1000, npcUnit: 'weiInfantry', npcUnitName: '魏步兵', npcAmount: 1000,
			hitPower: [10000, 8000], missPower: [10000, 10000], hitLosses: [421, 678], missLosses: [500, 600],
			hitTraits: [
				{ traitId: 'baibu_chuanyang', traitName: '百步穿杨', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'huangzhong', detail: { enemyDefenseReductionRate: 0.2, infantryDefenseModifiedUnits: { weiInfantry: -2 }, cavalryDefenseModifiedUnits: { weiInfantry: -2 }, triggerChance: 1 } },
				{ traitId: 'laodang_yizhuang', traitName: '老当益壮', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'huangzhong', detail: { effectRate: 0.1, extraLosses: { weiInfantry: 100 } } },
			],
			missTraits: [{ traitId: 'laodang_yizhuang', traitName: '老当益壮', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'huangzhong', detail: { effectRate: 0.1, extraLosses: { weiInfantry: 100 } } }], snapshot: {},
		},
		{
			generalId: 'luxun', generalName: '陆逊', traitIds: ['huoshao_lianying', 'lianying_zengshang'], playerUnit: 'weiInfantry', playerUnitName: '魏步兵', playerAmount: 1000, npcUnit: 'weiInfantry', npcUnitName: '魏步兵', npcAmount: 1000,
			hitPower: [10000, 10000], missPower: [10000, 10000], hitLosses: [500, 1000], missLosses: [500, 600],
			hitTraits: [{ traitId: 'huoshao_lianying', traitName: '火烧联营', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'luxun', detail: { effectRate: 1, maxAffectedRate: 1, targetExtraLosses: { weiInfantry: 500 }, triggerChance: 1 } }],
			missTraits: [{ traitId: 'lianying_zengshang', traitName: '连营增伤', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'luxun', detail: { effectRate: 0.1, targetExtraLosses: { weiInfantry: 100 } } }], snapshot: {},
		},
		{
			generalId: 'huanggai', generalName: '黄盖', traitIds: ['kurouji', 'kurou_fanji'], playerUnit: 'weiInfantry', playerUnitName: '魏步兵', playerAmount: 1000, npcUnit: 'weiInfantry', npcUnitName: '魏步兵', npcAmount: 1000,
			hitPower: [10000, 10000], missPower: [10000, 10000], hitLosses: [500, 600], missLosses: [500, 600],
			hitTraits: [
				{ traitId: 'kurouji', traitName: '苦肉计', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'huanggai', detail: { disableTraitCount: 1, disabledTraitCount: 0, triggerChance: 1 } },
				{ traitId: 'kurou_fanji', traitName: '苦肉反击', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'huanggai', detail: { effectRate: 0.1, extraLosses: { weiInfantry: 100 } } },
			],
			missTraits: [{ traitId: 'kurou_fanji', traitName: '苦肉反击', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'huanggai', detail: { effectRate: 0.1, extraLosses: { weiInfantry: 100 } } }], snapshot: {},
		},
		{
			generalId: 'sunce', generalName: '孙策', traitIds: ['xiaobawang_zhuiji', 'xiaobawang_tieqi'], playerUnit: 'overlordRider', playerUnitName: '霸王骑', playerAmount: 200, npcUnit: 'weiInfantry', npcUnitName: '魏步兵', npcAmount: 1000,
			hitPower: [15600, 8000], missPower: [15600, 8000], hitLosses: [55, 821], missLosses: [55, 721],
			hitTraits: [
				{ traitId: 'xiaobawang_tieqi', traitName: '小霸王', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'sunce', detail: { unitAttackFlat: 50, attackModifiedUnits: { overlordRider: 50 } } },
				{ traitId: 'xiaobawang_zhuiji', traitName: '小霸王追击', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'sunce', detail: { effectRate: 0.1, extraLosses: { weiInfantry: 100 }, triggerChance: 1 } },
			],
			missTraits: [{ traitId: 'xiaobawang_tieqi', traitName: '小霸王', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'sunce', detail: { unitAttackFlat: 50, attackModifiedUnits: { overlordRider: 50 } } }], snapshot: {},
		},
	]

	for (const current of cases) {
		for (const hit of [true, false]) {
			const powers = hit ? current.hitPower : current.missPower
			const losses = hit ? current.hitLosses : current.missLosses
			const traits = hit ? current.hitTraits : current.missTraits
			const detail = normalizeBattleReportDetail({
				id: `${current.generalId}-npc-late-${hit ? 'hit' : 'miss'}`, playerId: `player_${current.generalId}`, ownerPlayerId: `player_${current.generalId}`,
				viewType: 'attack', sourceType: 'npc_city', battleType: 'plunder', type: 'plunder', result: 'attacker_victory', rewards: {}, read: true, createdAt: '2026-07-21T02:15:00Z',
				detail: {
					id: `${current.generalId}-npc-late-${hit ? 'hit' : 'miss'}`, sourceType: 'npc_city', viewType: 'attack', battleType: 'plunder', result: 'attacker_victory', winnerSide: 'attacker', ownerSide: 'attacker',
					primarySide: {
						role: 'attacker', faction: 'wei', power: powers[0],
						generals: [{ id: current.generalId, name: current.generalName, level: 1, ...current.snapshot, traits: current.traitIds.map((traitId) => ({ traitId, name: getTraitMeta(traitId).name })) }],
						units: [{ unitType: current.playerUnit, unitName: current.playerUnitName, amountBefore: current.playerAmount, dispatched: current.playerAmount, lost: losses[0], survived: current.playerAmount - losses[0] }],
					},
					secondarySide: { role: 'defender', faction: 'wei', power: powers[1], units: [{ unitType: current.npcUnit, unitName: current.npcUnitName, amountBefore: current.npcAmount, dispatched: current.npcAmount, lost: losses[1], survived: current.npcAmount - losses[1] }] },
					rewards: { generalExp: losses[1], resources: {} }, traits,
				},
			})
			assert.equal(detail.sourceType, 'npc_city')
			assert.equal(detail.primarySide.power, powers[0])
			assert.equal(detail.secondarySide?.power, powers[1])
			assert.deepEqual(detail.primarySide.units[0], { unitType: current.playerUnit, unitName: current.playerUnitName, amountBefore: current.playerAmount, dispatched: current.playerAmount, lost: losses[0], survived: current.playerAmount - losses[0] })
			assert.deepEqual(detail.secondarySide?.units[0], { unitType: current.npcUnit, unitName: current.npcUnitName, amountBefore: current.npcAmount, dispatched: current.npcAmount, lost: losses[1], survived: current.npcAmount - losses[1] })
			assert.deepEqual(detail.primarySide.generals?.[0]?.traits?.map((trait) => trait.traitId), current.traitIds)
			assert.deepEqual(detail.traits?.map((trait) => trait.traitId), traits.map((trait) => trait.traitId))
			assert.equal(detail.rewards.generalExp, losses[1])
			assert.equal(detail.traits?.some((trait) => trait.traitId === current.traitIds[0]), hit)
		}
	}
})

test('赵云黄巾守城只展示龙胆真实减损且不补造七进七出', () => {
	const options = { faction: 'shu', units: { shu: { greedyWolf: { name: '贪狼营' } } } }
	const detail = {
		sourceType: 'yellow_turban', viewType: 'defense',
		secondarySide: {
			role: 'defender', generals: [{ id: 'zhaoyun', name: '赵云', traits: [{ traitId: 'longdan_jiuyuan' }, { traitId: 'qijin_qichu' }] }],
			units: [{ unitType: 'greedyWolf', unitName: '贪狼营', amountBefore: 100, dispatched: 100, lost: 80, survived: 20 }],
		},
		traits: [{
			traitId: 'longdan_jiuyuan', traitName: '龙胆救援', ownerSide: 'secondary', ownerRole: 'defender', generalId: 'zhaoyun',
			detail: { lossReductionRate: 0.2, reducedLosses: { greedyWolf: 20 }, triggerChance: 1 },
		}],
	}
	assert.deepEqual(detail.secondarySide.generals[0].traits.map((trait) => trait.traitId), ['longdan_jiuyuan', 'qijin_qichu'])
	assert.deepEqual(detail.traits.map((trait) => trait.traitId), ['longdan_jiuyuan'])
	assert.deepEqual(detail.traits.map((trait) => resolveReportTraitDisplaySide(detail, trait)), ['secondary'])
	assert.deepEqual(detail.secondarySide.units[0], { unitType: 'greedyWolf', unitName: '贪狼营', amountBefore: 100, dispatched: 100, lost: 80, survived: 20 })
	assert.equal(formatTraitOutcomeDetail('reducedLosses', detail.traits[0].detail.reducedLosses, options), '减少损失: 贪狼营 +20')
	assert.equal(getTraitMeta('qijin_qichu').trigger, '行军创建时')
})

test('关羽主动进攻双特性同时展示真实战前伤亡、加攻和最终兵力', () => {
	const detail = {
		sourceType: 'player_city', viewType: 'attack',
		primarySide: {
			role: 'attacker', power: 12000,
			units: [{ unitType: 'greedyWolf', unitName: '贪狼营', amountBefore: 1000, dispatched: 1000, lost: 436, survived: 564 }],
		},
		secondarySide: {
			role: 'defender', power: 6695,
			units: [{ unitType: 'qingZhouArmy', unitName: '青州军', amountBefore: 1000, dispatched: 1000, lost: 1000, survived: 0 }],
		},
		traits: [
			{
				traitId: 'shuiyan_qijun', traitName: '水淹七军', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'guanyu',
				detail: { effectRate: 0.35, maxAffectedRate: 0.35, preBattleAffected: { qingZhouArmy: 350 }, triggerChance: 1 },
			},
			{
				traitId: 'wusheng_pojun', traitName: '武圣破军', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'guanyu',
				detail: { attackBonusRate: 0.2, attackModifiedUnits: { greedyWolf: 2 } },
			},
		],
	}
	assert.equal(hasTraitEntries(detail), true)
	assert.deepEqual(detail.traits.map((trait) => trait.traitId), ['shuiyan_qijun', 'wusheng_pojun'])
	assert.deepEqual(detail.traits.map((trait) => getTraitMeta(trait.traitId).trigger), ['战斗前', '主动进攻战斗前'])
	assert.deepEqual(detail.traits.map((trait) => resolveReportTraitDisplaySide(detail, trait)), ['primary', 'primary'])
	assert.deepEqual(detail.primarySide.units[0], { unitType: 'greedyWolf', unitName: '贪狼营', amountBefore: 1000, dispatched: 1000, lost: 436, survived: 564 })
	assert.deepEqual(detail.secondarySide.units[0], { unitType: 'qingZhouArmy', unitName: '青州军', amountBefore: 1000, dispatched: 1000, lost: 1000, survived: 0 })
	assert.equal(formatTraitOutcomeDetail('preBattleAffected', detail.traits[0].detail.preBattleAffected), '战前真实伤亡: 青州军 +350')
	assert.equal(formatTraitOutcomeDetail('attackBonusRate', detail.traits[1].detail.attackBonusRate), '设计攻击加成: 20%')
	assert.equal(formatTraitOutcomeDetail('attackModifiedUnits', detail.traits[1].detail.attackModifiedUnits), '实际攻击修正: 贪狼营 +2')
})

test('关羽水淹未命中时武圣加攻仍生效且不产生战前伤亡', () => {
	const detail = {
		sourceType: 'player_city', viewType: 'attack', battleType: 'plunder',
		primarySide: {
			role: 'attacker', power: 12000,
			generals: [{
				id: 'guanyu', name: '关羽', level: 1,
				traits: [{ traitId: 'shuiyan_qijun', name: '水淹七军' }, { traitId: 'wusheng_pojun', name: '武圣破军' }],
			}],
			units: [{ unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 1000, dispatched: 1000, lost: 435, survived: 565 }],
		},
		secondarySide: {
			role: 'defender', power: 10000,
			units: [{ unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 1000, dispatched: 1000, lost: 564, survived: 436 }],
		},
		rewards: { generalExp: 564, resources: {} },
		traits: [{
			traitId: 'wusheng_pojun', traitName: '武圣破军', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'guanyu',
			detail: { attackBonusRate: 0.2, attackModifiedUnits: { shuInfantry: 2 } },
		}],
	}
	assert.deepEqual(detail.primarySide.generals[0].traits.map((trait) => trait.traitId), ['shuiyan_qijun', 'wusheng_pojun'])
	assert.deepEqual(detail.traits.map((trait) => trait.traitId), ['wusheng_pojun'])
	assert.equal(detail.traits.some((trait) => trait.traitId === 'shuiyan_qijun'), false)
	assert.deepEqual(detail.primarySide.units[0], { unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 1000, dispatched: 1000, lost: 435, survived: 565 })
	assert.deepEqual(detail.secondarySide.units[0], { unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 1000, dispatched: 1000, lost: 564, survived: 436 })
	assert.equal(formatTraitOutcomeDetail('attackBonusRate', detail.traits[0].detail.attackBonusRate), '设计攻击加成: 20%')
	assert.equal(formatTraitOutcomeDetail('attackModifiedUnits', detail.traits[0].detail.attackModifiedUnits, { faction: 'shu', units: { shu: { shuInfantry: { name: '蜀步兵' } } } }), '实际攻击修正: 蜀步兵 +2')
})

test('张飞主动进攻双特性区分本场压制、真实阵亡和最终存活', () => {
	const detail = {
		sourceType: 'player_city', viewType: 'attack',
		primarySide: {
			role: 'attacker', power: 12000,
			units: [{ unitType: 'greedyWolf', unitName: '贪狼营', amountBefore: 1000, dispatched: 1000, lost: 300, survived: 700 }],
		},
		secondarySide: {
			role: 'defender', power: 5150,
			units: [{ unitType: 'qingZhouArmy', unitName: '青州军', amountBefore: 1000, dispatched: 1000, lost: 500, survived: 500 }],
		},
		traits: [
			{
				traitId: 'zhenhe_quanjun', traitName: '震慑全军', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'zhangfei',
				detail: { effectRate: 0.5, maxAffectedRate: 0.5, suppressedUnits: { qingZhouArmy: 500 }, triggerChance: 1 },
			},
			{
				traitId: 'wanren_nuhou', traitName: '万人怒吼', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'zhangfei',
				detail: { attackBonusRate: 0.2, attackModifiedUnits: { greedyWolf: 2 } },
			},
		],
	}
	assert.equal(hasTraitEntries(detail), true)
	assert.deepEqual(detail.traits.map((trait) => trait.traitId), ['zhenhe_quanjun', 'wanren_nuhou'])
	assert.deepEqual(detail.traits.map((trait) => getTraitMeta(trait.traitId).trigger), ['战斗前', '主动进攻战斗前'])
	assert.deepEqual(detail.traits.map((trait) => resolveReportTraitDisplaySide(detail, trait)), ['primary', 'primary'])
	assert.deepEqual(detail.primarySide.units[0], { unitType: 'greedyWolf', unitName: '贪狼营', amountBefore: 1000, dispatched: 1000, lost: 300, survived: 700 })
	assert.deepEqual(detail.secondarySide.units[0], { unitType: 'qingZhouArmy', unitName: '青州军', amountBefore: 1000, dispatched: 1000, lost: 500, survived: 500 })
	assert.equal(formatTraitOutcomeDetail('suppressedUnits', detail.traits[0].detail.suppressedUnits), '本场压制兵力: 青州军 +500')
	assert.equal(formatTraitOutcomeDetail('attackBonusRate', detail.traits[1].detail.attackBonusRate), '设计攻击加成: 20%')
	assert.equal(formatTraitOutcomeDetail('attackModifiedUnits', detail.traits[1].detail.attackModifiedUnits), '实际攻击修正: 贪狼营 +2')
})

test('张飞震慑未命中时步兵加攻仍生效且不产生临时压制', () => {
	const detail = {
		sourceType: 'player_city', viewType: 'attack', battleType: 'attack',
		primarySide: {
			role: 'attacker', power: 12000,
			generals: [{
				id: 'zhangfei', name: '张飞', level: 1,
				traits: [{ traitId: 'zhenhe_quanjun', name: '震慑全军' }, { traitId: 'wanren_nuhou', name: '万人怒吼' }],
			}],
			units: [{ unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 1000, dispatched: 1000, lost: 804, survived: 196 }],
		},
		secondarySide: {
			role: 'defender', power: 10300,
			units: [{ unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 1000, dispatched: 1000, lost: 1000, survived: 0 }],
		},
		rewards: { generalExp: 1000, resources: {} },
		traits: [{
			traitId: 'wanren_nuhou', traitName: '万人怒吼', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'zhangfei',
			detail: { attackBonusRate: 0.2, attackModifiedUnits: { shuInfantry: 2 } },
		}],
	}
	assert.deepEqual(detail.primarySide.generals[0].traits.map((trait) => trait.traitId), ['zhenhe_quanjun', 'wanren_nuhou'])
	assert.deepEqual(detail.traits.map((trait) => trait.traitId), ['wanren_nuhou'])
	assert.equal(detail.traits.some((trait) => trait.traitId === 'zhenhe_quanjun'), false)
	assert.deepEqual(detail.primarySide.units[0], { unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 1000, dispatched: 1000, lost: 804, survived: 196 })
	assert.deepEqual(detail.secondarySide.units[0], { unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 1000, dispatched: 1000, lost: 1000, survived: 0 })
	assert.equal(formatTraitOutcomeDetail('attackBonusRate', detail.traits[0].detail.attackBonusRate), '设计攻击加成: 20%')
	assert.equal(formatTraitOutcomeDetail('attackModifiedUnits', detail.traits[0].detail.attackModifiedUnits, { faction: 'shu', units: { shu: { shuInfantry: { name: '蜀步兵' } } } }), '实际攻击修正: 蜀步兵 +2')
})

test('张辽主动进攻双特性对齐骑兵加攻、溃逃返回和最终战报数值', () => {
	const detail = {
		sourceType: 'player_city', viewType: 'attack',
		primarySide: {
			role: 'attacker', power: 19000,
			units: [{ unitType: 'weiCavalry', unitName: '魏骑兵', amountBefore: 1000, dispatched: 1000, lost: 199, survived: 801 }],
		},
		secondarySide: {
			role: 'defender', power: 6120,
			units: [{ unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 1000, dispatched: 1000, lost: 750, survived: 250 }],
		},
		traits: [
			{
				traitId: 'weizhen_zhenhe', traitName: '震慑全军', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'zhangliao',
				detail: { effectRate: 0.25, suppressedUnits: { shuInfantry: 250 }, fledUnits: { shuInfantry: 250 }, returnedUnits: { shuInfantry: 250 }, triggerChance: 1 },
			},
			{
				traitId: 'weizhen_xiaoyao', traitName: '威震逍遥', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'zhangliao',
				detail: { attackBonusRate: 0.35, attackModifiedUnits: { weiCavalry: 5 } },
			},
		],
	}
	const unitOptions = {
		faction: 'wei',
		units: { wei: { weiCavalry: { name: '魏骑兵' } }, shu: { shuInfantry: { name: '蜀步兵' } } },
	}
	assert.equal(hasTraitEntries(detail), true)
	assert.deepEqual(detail.traits.map((trait) => trait.traitId), ['weizhen_zhenhe', 'weizhen_xiaoyao'])
	assert.deepEqual(detail.traits.map((trait) => getTraitMeta(trait.traitId).trigger), ['主动进攻战斗前', '主动进攻战斗前'])
	assert.deepEqual(detail.traits.map((trait) => resolveReportTraitDisplaySide(detail, trait)), ['primary', 'primary'])
	assert.deepEqual(detail.primarySide, { role: 'attacker', power: 19000, units: [{ unitType: 'weiCavalry', unitName: '魏骑兵', amountBefore: 1000, dispatched: 1000, lost: 199, survived: 801 }] })
	assert.deepEqual(detail.secondarySide, { role: 'defender', power: 6120, units: [{ unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 1000, dispatched: 1000, lost: 750, survived: 250 }] })
	assert.equal(formatTraitOutcomeDetails(detail.traits[0].detail, unitOptions), '设计效果比例: 25%；本场溃逃兵力: 蜀步兵 +250；战后返回兵力: 蜀步兵 +250；触发概率: 100%')
	assert.equal(formatTraitOutcomeDetail('attackBonusRate', detail.traits[1].detail.attackBonusRate, unitOptions), '设计攻击加成: 35%')
	assert.equal(formatTraitOutcomeDetail('attackModifiedUnits', detail.traits[1].detail.attackModifiedUnits, unitOptions), '实际攻击修正: 魏骑兵 +5')
})

test('张辽震慑未命中时骑兵加攻仍生效且不产生临时压制', () => {
	const detail = {
		sourceType: 'player_city', viewType: 'attack', battleType: 'attack',
		primarySide: {
			role: 'attacker', power: 19000,
			generals: [{
				id: 'zhangliao', name: '张辽', level: 1,
				traits: [{ traitId: 'weizhen_zhenhe', name: '震慑全军' }, { traitId: 'weizhen_xiaoyao', name: '威震逍遥' }],
			}],
			units: [{ unitType: 'weiCavalry', unitName: '魏骑兵', amountBefore: 1000, dispatched: 1000, lost: 300, survived: 700 }],
		},
		secondarySide: {
			role: 'defender', power: 8160,
			units: [{ unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 1000, dispatched: 1000, lost: 1000, survived: 0 }],
		},
		rewards: { generalExp: 1000, resources: {} },
		traits: [{
			traitId: 'weizhen_xiaoyao', traitName: '威震逍遥', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'zhangliao',
			detail: { attackBonusRate: 0.35, attackModifiedUnits: { weiCavalry: 5 } },
		}],
	}
	assert.deepEqual(detail.primarySide.generals[0].traits.map((trait) => trait.traitId), ['weizhen_zhenhe', 'weizhen_xiaoyao'])
	assert.deepEqual(detail.traits.map((trait) => trait.traitId), ['weizhen_xiaoyao'])
	assert.equal(detail.traits.some((trait) => trait.traitId === 'weizhen_zhenhe'), false)
	assert.deepEqual(detail.primarySide.units[0], { unitType: 'weiCavalry', unitName: '魏骑兵', amountBefore: 1000, dispatched: 1000, lost: 300, survived: 700 })
	assert.deepEqual(detail.secondarySide.units[0], { unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 1000, dispatched: 1000, lost: 1000, survived: 0 })
	assert.equal(formatTraitOutcomeDetail('attackBonusRate', detail.traits[0].detail.attackBonusRate), '设计攻击加成: 35%')
	assert.equal(formatTraitOutcomeDetail('attackModifiedUnits', detail.traits[0].detail.attackModifiedUnits, { faction: 'wei', units: { wei: { weiCavalry: { name: '魏骑兵' } } } }), '实际攻击修正: 魏骑兵 +5')
})

test('关羽张辽张飞防守随机未命中时进攻专属加成也不越界触发', () => {
	const cases = [
		{ generalId: 'guanyu', generalName: '关羽', traitIds: ['shuiyan_qijun', 'wusheng_pojun'], attackerUnit: 'weiInfantry', attackerName: '魏步兵', defenderUnit: 'shuInfantry', defenderName: '蜀步兵' },
		{ generalId: 'zhangliao', generalName: '张辽', traitIds: ['weizhen_zhenhe', 'weizhen_xiaoyao'], attackerUnit: 'shuInfantry', attackerName: '蜀步兵', defenderUnit: 'weiInfantry', defenderName: '魏步兵' },
		{ generalId: 'zhangfei', generalName: '张飞', traitIds: ['zhenhe_quanjun', 'wanren_nuhou'], attackerUnit: 'weiInfantry', attackerName: '魏步兵', defenderUnit: 'shuInfantry', defenderName: '蜀步兵' },
	]
	for (const tc of cases) {
		const detail = {
			sourceType: 'player_city', viewType: 'defense', battleType: 'plunder', ownerSide: 'defender',
			primarySide: {
				role: 'attacker', power: 10000,
				units: [{ unitType: tc.attackerUnit, unitName: tc.attackerName, amountBefore: 1000, dispatched: 1000, lost: 500, survived: 500 }],
			},
			secondarySide: {
				role: 'defender', power: 10000,
				generals: [{ id: tc.generalId, name: tc.generalName, level: 1, traits: tc.traitIds.map((traitId) => ({ traitId, name: getTraitMeta(traitId).name })) }],
				units: [{ unitType: tc.defenderUnit, unitName: tc.defenderName, amountBefore: 1000, dispatched: 1000, lost: 500, survived: 500 }],
			},
			rewards: { generalExp: 500, resources: {} },
			traits: [],
		}
		assert.deepEqual(detail.secondarySide.generals[0].traits.map((trait) => trait.traitId), tc.traitIds)
		assert.equal(hasTraitEntries(detail), false)
		assert.deepEqual(detail.primarySide.units[0], { unitType: tc.attackerUnit, unitName: tc.attackerName, amountBefore: 1000, dispatched: 1000, lost: 500, survived: 500 })
		assert.deepEqual(detail.secondarySide.units[0], { unitType: tc.defenderUnit, unitName: tc.defenderName, amountBefore: 1000, dispatched: 1000, lost: 500, survived: 500 })
		assert.equal(getTraitMeta(tc.traitIds[0]).trigger, tc.generalId === 'zhangliao' ? '主动进攻战斗前' : '战斗前')
		assert.equal(getTraitMeta(tc.traitIds[1]).trigger, '主动进攻战斗前')
		assert.deepEqual(detail.rewards, { generalExp: 500, resources: {} })
	}
})

test('许褚虎痴命中与虎虎生威永久被动按不同区域展示', () => {
	const detail = {
		sourceType: 'player_city', viewType: 'attack',
		primarySide: {
			role: 'attacker', power: 8400,
			generals: [{
				id: 'xuchu', name: '许褚', level: 1,
				traits: [
					{ traitId: 'huchi_chongzhen', name: '虎痴冲阵', params: { triggerChance: 0.5, enemyDefenseReductionRate: 0.3 } },
					{ traitId: 'huhu_shengwei', name: '虎虎生威', params: { unitAttackFlat: 12, unitSpeedFlat: 5 } },
				],
			}],
			units: [{ unitType: 'huBaoQi', unitName: '虎豹骑', amountBefore: 200, dispatched: 200, lost: 100, survived: 100 }],
		},
		secondarySide: {
			role: 'defender', power: 7000,
			units: [{ unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 1000, dispatched: 1000, lost: 600, survived: 400 }],
		},
		traits: [{
			traitId: 'huchi_chongzhen', traitName: '虎痴冲阵', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'xuchu',
			detail: { enemyDefenseReductionRate: 0.3, infantryDefenseModifiedUnits: { shuInfantry: -3 }, cavalryDefenseModifiedUnits: { shuInfantry: -2 }, triggerChance: 0.5 },
		}],
	}
	const unitOptions = { faction: 'wei', units: { shu: { shuInfantry: { name: '蜀步兵' } } } }
	assert.deepEqual(detail.primarySide.generals[0].traits.map((trait) => trait.traitId), ['huchi_chongzhen', 'huhu_shengwei'])
	assert.deepEqual(detail.traits.map((trait) => trait.traitId), ['huchi_chongzhen'])
	assert.equal(detail.traits.some((trait) => trait.traitId === 'huhu_shengwei'), false)
	assert.equal(getTraitMeta('huchi_chongzhen').trigger, '主动进攻战斗前')
	assert.equal(getTraitMeta('huhu_shengwei').trigger, '永久被动')
	assert.deepEqual(detail.traits.map((trait) => resolveReportTraitDisplaySide(detail, trait)), ['primary'])
	assert.equal(formatTraitOutcomeDetail('enemyDefenseReductionRate', detail.traits[0].detail.enemyDefenseReductionRate, unitOptions), '设计敌方防御降低: 30%')
	assert.equal(formatTraitOutcomeDetail('infantryDefenseModifiedUnits', detail.traits[0].detail.infantryDefenseModifiedUnits, unitOptions), '实际步防修正: 蜀步兵 -3')
	assert.equal(formatTraitOutcomeDetail('cavalryDefenseModifiedUnits', detail.traits[0].detail.cavalryDefenseModifiedUnits, unitOptions), '实际骑防修正: 蜀步兵 -2')
	assert.equal(formatTraitOutcomeDetail('triggerChance', detail.traits[0].detail.triggerChance, unitOptions), '触发概率: 50%')
})

test('许褚虎痴合法未命中时不补造触发，虎虎生威仍保留在拥有快照', () => {
	const detail = {
		sourceType: 'player_city', viewType: 'attack',
		primarySide: {
			role: 'attacker', power: 8400,
			generals: [{
				id: 'xuchu', name: '许褚', level: 1,
				traits: [
					{ traitId: 'huchi_chongzhen', name: '虎痴冲阵', params: { triggerChance: 0.5, enemyDefenseReductionRate: 0.3 } },
					{ traitId: 'huhu_shengwei', name: '虎虎生威', params: { unitAttackFlat: 12, unitSpeedFlat: 5 } },
				],
			}],
			units: [{ unitType: 'huBaoQi', unitName: '虎豹骑', amountBefore: 200, dispatched: 200, lost: 120, survived: 80 }],
		},
		secondarySide: {
			role: 'defender', power: 10000,
			units: [{ unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 1000, dispatched: 1000, lost: 500, survived: 500 }],
		},
		traits: [],
	}
	assert.deepEqual(detail.primarySide.generals[0].traits.map((trait) => trait.traitId), ['huchi_chongzhen', 'huhu_shengwei'])
	assert.equal(hasTraitEntries(detail), false)
	assert.equal(detail.traits.some((trait) => trait.traitId === 'huchi_chongzhen'), false)
})


test('典韦双特性按攻防方向展示实际战前属性修正', () => {
	const detail = {
		sourceType: 'player_city', viewType: 'attack',
		primarySide: {
			role: 'attacker', power: 1400,
		units: [{ unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 100, dispatched: 100, lost: 94, survived: 6 }],
		},
		secondarySide: {
			role: 'defender', power: 10000,
		units: [{ unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 1000, dispatched: 1000, lost: 57, survived: 943 }],
		},
		traits: [
			{
				traitId: 'sizhandaodi', traitName: '死战到底', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'dianwei',
				detail: { attackBonusRate: 0.35, attackModifiedUnits: { weiInfantry: 4 } },
			},
		],
	}
	const unitOptions = { faction: 'wei', units: { wei: { weiInfantry: { name: '魏步兵' } } } }
	assert.deepEqual(detail.traits.map((trait) => trait.traitId), ['sizhandaodi'])
	assert.deepEqual(detail.traits.map((trait) => getTraitMeta(trait.traitId).trigger), ['主动进攻战斗前'])
	assert.deepEqual(detail.traits.map((trait) => resolveReportTraitDisplaySide(detail, trait)), ['primary'])
	assert.deepEqual(detail.primarySide, { role: 'attacker', power: 1400, units: [{ unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 100, dispatched: 100, lost: 94, survived: 6 }] })
	assert.deepEqual(detail.secondarySide, { role: 'defender', power: 10000, units: [{ unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 1000, dispatched: 1000, lost: 57, survived: 943 }] })
	assert.equal(formatTraitOutcomeDetail('attackBonusRate', detail.traits[0].detail.attackBonusRate, unitOptions), '设计攻击加成: 35%')
	assert.equal(formatTraitOutcomeDetail('attackModifiedUnits', detail.traits[0].detail.attackModifiedUnits, unitOptions), '实际攻击修正: 魏步兵 +4')
})

test('黄忠掠夺战双特性区分战前破防、核心战损和战后追加扣兵', () => {
	const detail = {
		sourceType: 'player_city', viewType: 'attack', battleType: 'plunder',
		primarySide: {
			role: 'attacker', power: 10000,
			units: [{ unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 1000, dispatched: 1000, lost: 421, survived: 579 }],
		},
		secondarySide: {
			role: 'defender', power: 8000,
			units: [{ unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 1000, dispatched: 1000, lost: 678, survived: 322 }],
		},
		traits: [
			{
				traitId: 'baibu_chuanyang', traitName: '百步穿杨', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'huangzhong',
				detail: { enemyDefenseReductionRate: 0.2, infantryDefenseModifiedUnits: { weiInfantry: -2 }, cavalryDefenseModifiedUnits: { weiInfantry: -2 }, triggerChance: 1 },
			},
			{
				traitId: 'laodang_yizhuang', traitName: '老当益壮', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'huangzhong',
				detail: { effectRate: 0.1, extraLosses: { weiInfantry: 100 } },
			},
		],
	}
	const unitOptions = { faction: 'shu', units: { wei: { weiInfantry: { name: '魏步兵' } } } }
	assert.deepEqual(detail.traits.map((trait) => trait.traitId), ['baibu_chuanyang', 'laodang_yizhuang'])
	assert.deepEqual(detail.traits.map((trait) => getTraitMeta(trait.traitId).trigger), ['主动进攻战斗前', '战斗结算后'])
	assert.deepEqual(detail.traits.map((trait) => resolveReportTraitDisplaySide(detail, trait)), ['primary', 'primary'])
	assert.deepEqual(detail.primarySide, { role: 'attacker', power: 10000, units: [{ unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 1000, dispatched: 1000, lost: 421, survived: 579 }] })
	assert.deepEqual(detail.secondarySide, { role: 'defender', power: 8000, units: [{ unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 1000, dispatched: 1000, lost: 678, survived: 322 }] })
	assert.equal(formatTraitOutcomeDetail('enemyDefenseReductionRate', detail.traits[0].detail.enemyDefenseReductionRate, unitOptions), '设计敌方防御降低: 20%')
	assert.equal(formatTraitOutcomeDetail('infantryDefenseModifiedUnits', detail.traits[0].detail.infantryDefenseModifiedUnits, unitOptions), '实际步防修正: 魏步兵 -2')
	assert.equal(formatTraitOutcomeDetail('cavalryDefenseModifiedUnits', detail.traits[0].detail.cavalryDefenseModifiedUnits, unitOptions), '实际骑防修正: 魏步兵 -2')
	assert.equal(formatTraitOutcomeDetail('effectRate', detail.traits[1].detail.effectRate, unitOptions), '设计效果比例: 10%')
	assert.equal(formatTraitOutcomeDetail('extraLosses', detail.traits[1].detail.extraLosses, unitOptions), '追加损失: 魏步兵 +100')
})

test('黄忠百步未命中时老当益壮仍按核心剩余兵力追加真实损失', () => {
	const detail = {
		sourceType: 'player_city', viewType: 'attack', battleType: 'plunder',
		primarySide: {
			role: 'attacker', power: 10000,
			generals: [{
				id: 'huangzhong', name: '黄忠', level: 1,
				traits: [{ traitId: 'baibu_chuanyang', name: '百步穿杨' }, { traitId: 'laodang_yizhuang', name: '老当益壮' }],
			}],
			units: [{ unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 1000, dispatched: 1000, lost: 500, survived: 500 }],
		},
		secondarySide: {
			role: 'defender', power: 10000,
			units: [{ unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 1000, dispatched: 1000, lost: 600, survived: 400 }],
		},
		rewards: { generalExp: 600, resources: {} },
		traits: [{
			traitId: 'laodang_yizhuang', traitName: '老当益壮', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'huangzhong',
			detail: { effectRate: 0.1, extraLosses: { weiInfantry: 100 } },
		}],
	}
	const unitOptions = { faction: 'shu', units: { wei: { weiInfantry: { name: '魏步兵' } } } }
	assert.deepEqual(detail.primarySide.generals[0].traits.map((trait) => trait.traitId), ['baibu_chuanyang', 'laodang_yizhuang'])
	assert.deepEqual(detail.traits.map((trait) => trait.traitId), ['laodang_yizhuang'])
	assert.equal(detail.traits.some((trait) => trait.traitId === 'baibu_chuanyang'), false)
	assert.deepEqual(detail.primarySide.units[0], { unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 1000, dispatched: 1000, lost: 500, survived: 500 })
	assert.deepEqual(detail.secondarySide.units[0], { unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 1000, dispatched: 1000, lost: 600, survived: 400 })
	assert.equal(formatTraitOutcomeDetail('effectRate', detail.traits[0].detail.effectRate, unitOptions), '设计效果比例: 10%')
	assert.equal(formatTraitOutcomeDetail('extraLosses', detail.traits[0].detail.extraLosses, unitOptions), '追加损失: 魏步兵 +100')
})

test('黄忠战后伤害完整汇总同兵种主守军与援军且战报显示追加一百', () => {
	const report = {
		id: 'huangzhong-coalition-damage', playerId: 'attacker', ownerPlayerId: 'attacker', viewType: 'attack', sourceType: 'player_city',
		battleType: 'plunder', type: 'attack', result: 'draw', defenderRevealed: true, rewards: {}, read: true, createdAt: '2026-07-20T12:00:00Z',
		pvpReinforcements: [{
			reinforcementId: 'rein_taishici', fromPlayerId: 'helper', fromPlayerName: '太史慈援军', faction: 'wu',
			troops: { wuInfantry: 500 }, generalExpGained: 500, generals: [{ id: 'taishici', name: '太史慈', level: 1 }],
		}],
		pvpReinforcementLosses: { rein_taishici: { wuInfantry: 300 } },
		detail: {
			id: 'huangzhong-coalition-damage', sourceType: 'player_city', viewType: 'attack', battleType: 'plunder', result: 'draw',
			primarySide: {
				role: 'attacker', power: 10000, generals: [{ id: 'huangzhong', name: '黄忠', level: 1 }],
				units: [{ unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 1000, dispatched: 1000, lost: 500, survived: 500 }],
			},
			secondarySide: {
				role: 'defender', power: 10000, generals: [{ id: 'sunquan', name: '孙权', level: 1 }],
				units: [{ unitType: 'wuInfantry', unitName: '吴步兵', amountBefore: 500, dispatched: 500, lost: 300, survived: 200 }],
			},
			rewards: { generalExp: 600, resources: {} },
			traits: [{
				traitId: 'laodang_yizhuang', traitName: '老当益壮', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'huangzhong',
				detail: { effectRate: 0.1, extraLosses: { wuInfantry: 100 }, triggerChance: 1 },
			}],
		},
	}
	const detail = normalizeBattleReportDetail(report)
	const pvp = detail.extra?.pvp && typeof detail.extra.pvp === 'object' ? detail.extra.pvp : undefined
	const reinforcement = pvp?.reinforcements[0]
	const options = { faction: 'wu', units: { wu: { wuInfantry: { name: '吴步兵' } } } }

	assert.deepEqual(detail.primarySide.units[0], { unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 1000, dispatched: 1000, lost: 500, survived: 500 })
	assert.deepEqual(detail.secondarySide?.units[0], { unitType: 'wuInfantry', unitName: '吴步兵', amountBefore: 500, dispatched: 500, lost: 300, survived: 200 })
	assert.equal(detail.rewards?.generalExp, 600)
	assert.deepEqual(detail.rewards?.resources, {})
	assert.equal(reinforcement?.generalExpGained, 500)
	assert.deepEqual(reinforcement?.troops, { wuInfantry: 500 })
	assert.deepEqual(pvp?.reinforcementLosses, { rein_taishici: { wuInfantry: 300 } })
	assert.deepEqual(detail.traits?.map((trait) => trait.traitId), ['laodang_yizhuang'])
	assert.equal(formatTraitOutcomeDetail('extraLosses', detail.traits?.[0].detail.extraLosses, options), '追加损失: 吴步兵 +100')
})

test('孙策掠夺战双特性只强化霸王骑并在胜利后追加真实追击损失', () => {
	const detail = {
		sourceType: 'player_city', viewType: 'attack', battleType: 'plunder',
		primarySide: {
			role: 'attacker', power: 15600,
			units: [{ unitType: 'overlordRider', unitName: '霸王骑', amountBefore: 200, dispatched: 200, lost: 55, survived: 145 }],
		},
		secondarySide: {
			role: 'defender', power: 8000,
			units: [{ unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 1000, dispatched: 1000, lost: 821, survived: 179 }],
		},
		traits: [
			{
				traitId: 'xiaobawang_tieqi', traitName: '小霸王', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'sunce',
				detail: { unitAttackFlat: 50, attackModifiedUnits: { overlordRider: 50 } },
			},
			{
				traitId: 'xiaobawang_zhuiji', traitName: '小霸王追击', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'sunce',
				detail: { effectRate: 0.1, extraLosses: { weiInfantry: 100 }, triggerChance: 1 },
			},
		],
	}
	const unitOptions = { faction: 'wu', units: { wu: { overlordRider: { name: '霸王骑' } }, wei: { weiInfantry: { name: '魏步兵' } } } }
	assert.deepEqual(detail.traits.map((trait) => trait.traitId), ['xiaobawang_tieqi', 'xiaobawang_zhuiji'])
	assert.deepEqual(detail.traits.map((trait) => getTraitMeta(trait.traitId).trigger), ['主动进攻战斗前', '掠夺战结算后'])
	assert.deepEqual(detail.traits.map((trait) => resolveReportTraitDisplaySide(detail, trait)), ['primary', 'primary'])
	assert.deepEqual(detail.primarySide, { role: 'attacker', power: 15600, units: [{ unitType: 'overlordRider', unitName: '霸王骑', amountBefore: 200, dispatched: 200, lost: 55, survived: 145 }] })
	assert.deepEqual(detail.secondarySide, { role: 'defender', power: 8000, units: [{ unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 1000, dispatched: 1000, lost: 821, survived: 179 }] })
	assert.equal(formatTraitOutcomeDetail('unitAttackFlat', detail.traits[0].detail.unitAttackFlat, unitOptions), '设计单位攻击增加: 50')
	assert.equal(formatTraitOutcomeDetail('attackModifiedUnits', detail.traits[0].detail.attackModifiedUnits, unitOptions), '实际攻击修正: 霸王骑 +50')
	assert.equal(formatTraitOutcomeDetail('effectRate', detail.traits[1].detail.effectRate, unitOptions), '设计效果比例: 10%')
	assert.equal(formatTraitOutcomeDetail('extraLosses', detail.traits[1].detail.extraLosses, unitOptions), '追加损失: 魏步兵 +100')
})

test('孙策掠夺获胜但追击未命中时只展示铁骑加攻和核心兵损', () => {
	const detail = {
		sourceType: 'player_city', viewType: 'attack', battleType: 'plunder',
		primarySide: {
			role: 'attacker', power: 15600,
			generals: [{
				id: 'sunce', name: '孙策', level: 1,
				traits: [{ traitId: 'xiaobawang_zhuiji', name: '小霸王追击' }, { traitId: 'xiaobawang_tieqi', name: '小霸王' }],
			}],
			units: [{ unitType: 'overlordRider', unitName: '霸王骑', amountBefore: 200, dispatched: 200, lost: 55, survived: 145 }],
		},
		secondarySide: {
			role: 'defender', power: 8000,
			units: [{ unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 1000, dispatched: 1000, lost: 721, survived: 279 }],
		},
		rewards: { generalExp: 721, resources: { wood: 10000 } },
		traits: [{
			traitId: 'xiaobawang_tieqi', traitName: '小霸王', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'sunce',
			detail: { unitAttackFlat: 50, attackModifiedUnits: { overlordRider: 50 } },
		}],
	}
	assert.deepEqual(detail.primarySide.generals[0].traits.map((trait) => trait.traitId), ['xiaobawang_zhuiji', 'xiaobawang_tieqi'])
	assert.deepEqual(detail.traits.map((trait) => trait.traitId), ['xiaobawang_tieqi'])
	assert.equal(detail.traits.some((trait) => trait.traitId === 'xiaobawang_zhuiji'), false)
	assert.deepEqual(detail.primarySide.units[0], { unitType: 'overlordRider', unitName: '霸王骑', amountBefore: 200, dispatched: 200, lost: 55, survived: 145 })
	assert.deepEqual(detail.secondarySide.units[0], { unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 1000, dispatched: 1000, lost: 721, survived: 279 })
	assert.equal(formatTraitOutcomeDetail('attackModifiedUnits', detail.traits[0].detail.attackModifiedUnits, { faction: 'wu', units: { wu: { overlordRider: { name: '霸王骑' } } } }), '实际攻击修正: 霸王骑 +50')
})

test('马超孙策防守随机条件成立但未命中时只展示核心结算', () => {
	const cases = [
		{
			general: { id: 'machao', name: '马超', level: 1, stats: { force: 0 }, effectiveStats: { force: 20 }, buffs: { attackBonus: 0.4 }, traits: [{ traitId: 'xiliang_tuji', name: '西凉突击' }, { traitId: 'tianshen_xiafan', name: '天神下凡' }] },
			primarySide: { role: 'attacker', power: 20000, units: [{ unitType: 'wuInfantry', unitName: '吴步兵', amountBefore: 1000, dispatched: 1000, lost: 12, survived: 988 }, { unitType: 'wuCavalry', unitName: '吴骑兵', amountBefore: 1000, dispatched: 1000, lost: 12, survived: 988 }] },
			secondaryPower: 900, secondaryUnit: { unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 100, dispatched: 100, lost: 100, survived: 0 }, generalExp: 24,
		},
		{
			general: { id: 'sunce', name: '孙策', level: 1, traits: [{ traitId: 'xiaobawang_zhuiji', name: '小霸王追击' }, { traitId: 'xiaobawang_tieqi', name: '小霸王' }] },
			primarySide: { role: 'attacker', power: 1000, units: [{ unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 100, dispatched: 100, lost: 72, survived: 28 }] },
			secondaryPower: 2000, secondaryUnit: { unitType: 'wuInfantry', unitName: '吴步兵', amountBefore: 200, dispatched: 200, lost: 54, survived: 146 }, generalExp: 72,
		},
	]
	for (const tc of cases) {
		const detail = {
			sourceType: 'player_city', viewType: 'defense', battleType: tc.general.id === 'sunce' ? 'plunder' : 'attack', ownerSide: 'defender',
			primarySide: tc.primarySide,
			secondarySide: { role: 'defender', power: tc.secondaryPower, generals: [tc.general], units: [tc.secondaryUnit] },
			rewards: { generalExp: tc.generalExp, resources: {} }, traits: [],
		}
		assert.deepEqual(detail.secondarySide.generals[0].traits.map((trait) => trait.traitId), tc.general.traits.map((trait) => trait.traitId))
		assert.equal(hasTraitEntries(detail), false)
		assert.deepEqual(detail.primarySide, tc.primarySide)
		assert.deepEqual(detail.secondarySide.units[0], tc.secondaryUnit)
		assert.deepEqual(detail.rewards, { generalExp: tc.generalExp, resources: {} })
		if (tc.general.id === 'machao') {
			assert.equal(tc.general.effectiveStats.force - tc.general.stats.force, 20)
			assert.equal(tc.general.buffs.attackBonus, 0.4)
		}
	}
})

test('孙策掠夺平局时拥有追击但不追加伤亡或生成时间线', () => {
	const detail = {
		sourceType: 'player_city', viewType: 'attack', battleType: 'plunder', result: 'draw', winnerSide: 'none',
		primarySide: {
			role: 'attacker', faction: 'wu', power: 1000,
			generals: [{
				id: 'sunce', name: '孙策', level: 1,
				traits: [{ traitId: 'xiaobawang_zhuiji', name: '小霸王追击', requiredOutcome: 'win' }, { traitId: 'xiaobawang_tieqi', name: '小霸王' }],
			}],
			units: [{ unitType: 'wuInfantry', unitName: '吴步兵', amountBefore: 100, dispatched: 100, lost: 50, survived: 50 }],
		},
		secondarySide: {
			role: 'defender', faction: 'shu', power: 1000,
			units: [{ unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 100, dispatched: 100, lost: 50, survived: 50 }],
		},
		rewards: { generalExp: 50, resources: {} }, traits: [],
	}
	assert.deepEqual(detail.primarySide.generals[0].traits.map((trait) => trait.traitId), ['xiaobawang_zhuiji', 'xiaobawang_tieqi'])
	assert.equal(hasTraitEntries(detail), false)
	assert.deepEqual(detail.primarySide.units[0], { unitType: 'wuInfantry', unitName: '吴步兵', amountBefore: 100, dispatched: 100, lost: 50, survived: 50 })
	assert.deepEqual(detail.secondarySide.units[0], { unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 100, dispatched: 100, lost: 50, survived: 50 })
	assert.equal(detail.rewards.generalExp, 50)
})

test('孙策掠夺守城获胜后追击来袭方并归属防守侧', () => {
	const detail = {
		sourceType: 'player_city', viewType: 'defense', battleType: 'plunder', result: 'defender_victory', winnerSide: 'defender', ownerSide: 'defender',
		primarySide: {
			role: 'attacker', faction: 'wei', power: 1000,
			units: [{ unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 100, dispatched: 100, lost: 100, survived: 0 }],
		},
		secondarySide: {
			role: 'defender', faction: 'wu', power: 2000,
			generals: [{ id: 'sunce', name: '孙策', level: 1, traits: [{ traitId: 'xiaobawang_zhuiji', name: '小霸王追击', requiredOutcome: 'win' }] }],
			units: [{ unitType: 'wuInfantry', unitName: '吴步兵', amountBefore: 200, dispatched: 200, lost: 54, survived: 146 }],
		},
		rewards: { generalExp: 100, resources: {} },
		traits: [{
			traitId: 'xiaobawang_zhuiji', traitName: '小霸王追击', ownerSide: 'secondary', ownerRole: 'defender', generalId: 'sunce',
			detail: { effectRate: 0.5, extraLosses: { weiInfantry: 28 }, triggerChance: 1 },
		}],
	}
	assert.deepEqual(detail.primarySide.units[0], { unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 100, dispatched: 100, lost: 100, survived: 0 })
	assert.deepEqual(detail.secondarySide.units[0], { unitType: 'wuInfantry', unitName: '吴步兵', amountBefore: 200, dispatched: 200, lost: 54, survived: 146 })
	assert.equal(detail.secondarySide.generals[0].traits[0].traitId, 'xiaobawang_zhuiji')
	assert.equal(resolveReportTraitDisplaySide(detail, detail.traits[0]), 'secondary')
	assert.equal(formatTraitOutcomeDetail('extraLosses', detail.traits[0].detail.extraLosses, { faction: 'wei', units: { wei: { weiInfantry: { name: '魏步兵' } } } }), '追加损失: 魏步兵 +28')
	assert.equal(detail.rewards.generalExp, 100)
	assert.equal(hasTraitEntries(detail), true)
})

test('孙策援军只在掠夺守方获胜且命中时展示真实追击', () => {
	const buildReport = (triggered: boolean) => ({
		id: `sunce-reinforcement-${triggered ? 'hit' : 'miss'}`, playerId: 'attacker', ownerPlayerId: 'attacker', viewType: 'attack', sourceType: 'player_city',
		battleType: 'plunder', type: 'attack', result: 'defender_victory', defenderRevealed: true, rewards: {}, read: true, createdAt: '2026-07-20T16:00:00Z',
		pvpReinforcements: [{
			reinforcementId: 'rein_sunce', fromPlayerId: 'helper_sunce', fromPlayerName: '孙策援军', faction: 'wu',
			troops: { wuInfantry: 199 }, generalExpGained: triggered ? 82 : 72,
			generals: [{ id: 'sunce', name: '孙策', level: 1, traits: [{ traitId: 'xiaobawang_zhuiji', name: '小霸王追击', requiredOutcome: 'win' }] }],
		}],
		pvpReinforcementLosses: { rein_sunce: { wuInfantry: 54 } },
		detail: {
			id: `sunce-reinforcement-${triggered ? 'hit' : 'miss'}`, sourceType: 'player_city', viewType: 'attack', battleType: 'plunder', result: 'defender_victory', winnerSide: 'defender',
			primarySide: {
				role: 'attacker', faction: 'wei', power: 1000, generals: [{ id: 'caocao', name: '曹操', level: 1 }],
				units: [{ unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 100, dispatched: 100, lost: triggered ? 82 : 72, survived: triggered ? 18 : 28 }],
			},
			secondarySide: {
				role: 'defender', faction: 'shu', power: 2000, generals: [{ id: 'liubei', name: '刘备', level: 1 }],
				units: [{ unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 1, dispatched: 1, lost: 0, survived: 1 }],
			},
			rewards: { generalExp: 54, resources: {} },
			traits: triggered ? [{
				traitId: 'xiaobawang_zhuiji', traitName: '小霸王追击', ownerSide: 'reinforcement', ownerRole: 'reinforcement', ownerPlayerId: 'helper_sunce', generalId: 'sunce',
				detail: { effectRate: 0.1, extraLosses: { weiInfantry: 10 }, triggerChance: 1 },
			}] : [],
		},
	})
	const hit = normalizeBattleReportDetail(buildReport(true))
	const miss = normalizeBattleReportDetail(buildReport(false))
	const hitPvp = hit.extra?.pvp && typeof hit.extra.pvp === 'object' ? hit.extra.pvp : undefined
	const missPvp = miss.extra?.pvp && typeof miss.extra.pvp === 'object' ? miss.extra.pvp : undefined
	const options = { faction: 'wei', units: { wei: { weiInfantry: { name: '魏步兵' } } } }

	assert.deepEqual(hit.primarySide.units[0], { unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 100, dispatched: 100, lost: 82, survived: 18 })
	assert.deepEqual(miss.primarySide.units[0], { unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 100, dispatched: 100, lost: 72, survived: 28 })
	assert.deepEqual(hit.secondarySide?.units[0], { unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 1, dispatched: 1, lost: 0, survived: 1 })
	assert.deepEqual(hitPvp?.reinforcementLosses, { rein_sunce: { wuInfantry: 54 } })
	assert.equal(hitPvp?.reinforcements[0]?.generalExpGained, 82)
	assert.equal(missPvp?.reinforcements[0]?.generalExpGained, 72)
	assert.deepEqual(hitPvp?.reinforcements[0]?.generals?.[0]?.traits?.map((trait) => trait.traitId), ['xiaobawang_zhuiji'])
	assert.deepEqual(missPvp?.reinforcements[0]?.generals?.[0]?.traits?.map((trait) => trait.traitId), ['xiaobawang_zhuiji'])
	assert.deepEqual(hit.traits?.map((trait) => trait.traitId), ['xiaobawang_zhuiji'])
	assert.equal(resolveReportTraitDisplaySide(hit, hit.traits?.[0]), 'reinforcement')
	assert.equal(formatTraitOutcomeDetail('extraLosses', hit.traits?.[0].detail.extraLosses, options), '追加损失: 魏步兵 +10')
	assert.equal(hasTraitEntries(miss), false)
})

test('陆逊援军火烧命中或未命中时只展示实际生效的追加伤害', () => {
	const buildReport = (fireTriggered: boolean) => ({
		id: `luxun-reinforcement-${fireTriggered ? 'hit' : 'miss'}`, playerId: 'attacker', ownerPlayerId: 'attacker', viewType: 'attack', sourceType: 'player_city',
		battleType: 'plunder', type: 'attack', result: 'draw', defenderRevealed: true, rewards: {}, read: true, createdAt: '2026-07-20T17:00:00Z',
		pvpReinforcements: [{
			reinforcementId: 'rein_luxun', fromPlayerId: 'helper_luxun', fromPlayerName: '陆逊援军', faction: 'wu',
			troops: { wuInfantry: 99 }, generalExpGained: fireTriggered ? 100 : 60,
			generals: [{
				id: 'luxun', name: '陆逊', level: 1,
				traits: [{ traitId: 'huoshao_lianying', name: '火烧联营' }, { traitId: 'lianying_zengshang', name: '连营增伤' }],
			}],
		}],
		pvpReinforcementLosses: { rein_luxun: { wuInfantry: 49 } },
		detail: {
			id: `luxun-reinforcement-${fireTriggered ? 'hit' : 'miss'}`, sourceType: 'player_city', viewType: 'attack', battleType: 'plunder', result: 'draw', winnerSide: 'none',
			primarySide: {
				role: 'attacker', faction: 'wei', power: 1000, generals: [{ id: 'caocao', name: '曹操', level: 1 }],
				units: [{ unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 100, dispatched: 100, lost: fireTriggered ? 100 : 60, survived: fireTriggered ? 0 : 40 }],
			},
			secondarySide: {
				role: 'defender', faction: 'shu', power: 1000, generals: [{ id: 'liubei', name: '刘备', level: 1 }],
				units: [{ unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 1, dispatched: 1, lost: 1, survived: 0 }],
			},
			rewards: { generalExp: 50, resources: {} },
			traits: [{
				traitId: fireTriggered ? 'huoshao_lianying' : 'lianying_zengshang', traitName: fireTriggered ? '火烧联营' : '连营增伤',
				ownerSide: 'reinforcement', ownerRole: 'reinforcement', ownerPlayerId: 'helper_luxun', generalId: 'luxun',
				detail: fireTriggered
					? { effectRate: 1, maxAffectedRate: 1, targetExtraLosses: { weiInfantry: 50 }, triggerChance: 1 }
					: { effectRate: 0.1, targetExtraLosses: { weiInfantry: 10 } },
			}],
		},
	})
	const hit = normalizeBattleReportDetail(buildReport(true))
	const miss = normalizeBattleReportDetail(buildReport(false))
	const hitPvp = hit.extra?.pvp && typeof hit.extra.pvp === 'object' ? hit.extra.pvp : undefined
	const missPvp = miss.extra?.pvp && typeof miss.extra.pvp === 'object' ? miss.extra.pvp : undefined
	const options = { faction: 'wei', units: { wei: { weiInfantry: { name: '魏步兵' } } } }

	assert.deepEqual(hit.primarySide.units[0], { unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 100, dispatched: 100, lost: 100, survived: 0 })
	assert.deepEqual(miss.primarySide.units[0], { unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 100, dispatched: 100, lost: 60, survived: 40 })
	assert.deepEqual(hit.secondarySide?.units[0], { unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 1, dispatched: 1, lost: 1, survived: 0 })
	assert.deepEqual(hitPvp?.reinforcementLosses, { rein_luxun: { wuInfantry: 49 } })
	assert.equal(hitPvp?.reinforcements[0]?.generalExpGained, 100)
	assert.equal(missPvp?.reinforcements[0]?.generalExpGained, 60)
	for (const pvp of [hitPvp, missPvp]) {
		assert.deepEqual(pvp?.reinforcements[0]?.generals?.[0]?.traits?.map((trait) => trait.traitId), ['huoshao_lianying', 'lianying_zengshang'])
	}
	assert.deepEqual(hit.traits?.map((trait) => trait.traitId), ['huoshao_lianying'])
	assert.deepEqual(miss.traits?.map((trait) => trait.traitId), ['lianying_zengshang'])
	assert.equal(resolveReportTraitDisplaySide(hit, hit.traits?.[0]), 'reinforcement')
	assert.equal(resolveReportTraitDisplaySide(miss, miss.traits?.[0]), 'reinforcement')
	assert.equal(formatTraitOutcomeDetail('targetExtraLosses', hit.traits?.[0].detail.targetExtraLosses, options), '目标兵种追加损失: 魏步兵 +50')
	assert.equal(formatTraitOutcomeDetail('targetExtraLosses', miss.traits?.[0].detail.targetExtraLosses, options), '目标兵种追加损失: 魏步兵 +10')
})

test('司马懿援军双特性同时命中或均未命中时准确展示', () => {
	const buildReport = (triggered: boolean) => ({
		id: `simayi-reinforcement-${triggered ? 'hit' : 'miss'}`, playerId: 'attacker', ownerPlayerId: 'attacker', viewType: 'attack', sourceType: 'player_city',
		battleType: 'plunder', type: 'attack', result: triggered ? 'defender_victory' : 'draw', defenderRevealed: true, rewards: {}, read: true, createdAt: '2026-07-20T18:00:00Z',
		pvpReinforcements: [{
			reinforcementId: 'rein_simayi', fromPlayerId: 'helper_simayi', fromPlayerName: '司马懿援军', faction: 'wei',
			troops: { weiInfantry: 99 }, generalExpGained: triggered ? 77 : 50,
			generals: [{
				id: 'simayi', name: '司马懿', level: 1,
				traits: [{ traitId: 'yibing_touxi', name: '疑兵偷袭' }, { traitId: 'mouding_houfa', name: '谋定后发', allowedSides: ['defender', 'reinforcement'] }],
			}],
		}],
		pvpReinforcementLosses: { rein_simayi: { weiInfantry: triggered ? 35 : 49 } },
		detail: {
			id: `simayi-reinforcement-${triggered ? 'hit' : 'miss'}`, sourceType: 'player_city', viewType: 'attack', battleType: 'plunder',
			result: triggered ? 'defender_victory' : 'draw', winnerSide: triggered ? 'defender' : 'none',
			primarySide: {
				role: 'attacker', faction: 'shu', power: triggered ? 650 : 1000, generals: [{ id: 'liubei', name: '刘备', level: 1 }],
				units: [{ unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 100, dispatched: 100, lost: triggered ? 77 : 50, survived: triggered ? 23 : 50 }],
			},
			secondarySide: {
				role: 'defender', faction: 'wu', power: triggered ? 1396 : 1000, generals: [{ id: 'sunquan', name: '孙权', level: 1 }],
				units: [{ unitType: 'wuInfantry', unitName: '吴步兵', amountBefore: 1, dispatched: 1, lost: triggered ? 0 : 1, survived: triggered ? 1 : 0 }],
			},
			rewards: { generalExp: triggered ? 35 : 50, resources: {} },
			traits: triggered ? [
				{
					traitId: 'yibing_touxi', traitName: '疑兵偷袭', ownerSide: 'reinforcement', ownerRole: 'reinforcement', ownerPlayerId: 'helper_simayi', generalId: 'simayi',
					detail: { effectRate: 0.35, preBattleAffected: { shuInfantry: 35 }, triggerChance: 1 },
				},
				{
					traitId: 'mouding_houfa', traitName: '谋定后发', ownerSide: 'reinforcement', ownerRole: 'reinforcement', ownerPlayerId: 'helper_simayi', generalId: 'simayi',
					detail: { defenseBonusRate: 0.35, infantryDefenseModifiedUnits: { weiInfantry: 4 }, cavalryDefenseModifiedUnits: { weiInfantry: 3 }, triggerChance: 1 },
				},
			] : [],
		},
	})
	const hit = normalizeBattleReportDetail(buildReport(true))
	const miss = normalizeBattleReportDetail(buildReport(false))
	const hitPvp = hit.extra?.pvp && typeof hit.extra.pvp === 'object' ? hit.extra.pvp : undefined
	const missPvp = miss.extra?.pvp && typeof miss.extra.pvp === 'object' ? miss.extra.pvp : undefined
	const options = { faction: 'shu', units: { shu: { shuInfantry: { name: '蜀步兵' } } } }

	assert.deepEqual(hit.primarySide.units[0], { unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 100, dispatched: 100, lost: 77, survived: 23 })
	assert.deepEqual(miss.primarySide.units[0], { unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 100, dispatched: 100, lost: 50, survived: 50 })
	assert.deepEqual(hit.secondarySide?.units[0], { unitType: 'wuInfantry', unitName: '吴步兵', amountBefore: 1, dispatched: 1, lost: 0, survived: 1 })
	assert.deepEqual(miss.secondarySide?.units[0], { unitType: 'wuInfantry', unitName: '吴步兵', amountBefore: 1, dispatched: 1, lost: 1, survived: 0 })
	assert.deepEqual(hitPvp?.reinforcementLosses, { rein_simayi: { weiInfantry: 35 } })
	assert.deepEqual(missPvp?.reinforcementLosses, { rein_simayi: { weiInfantry: 49 } })
	assert.equal(hitPvp?.reinforcements[0]?.generalExpGained, 77)
	assert.equal(missPvp?.reinforcements[0]?.generalExpGained, 50)
	for (const pvp of [hitPvp, missPvp]) {
		assert.deepEqual(pvp?.reinforcements[0]?.generals?.[0]?.traits?.map((trait) => trait.traitId), ['yibing_touxi', 'mouding_houfa'])
	}
	assert.deepEqual(hit.traits?.map((trait) => trait.traitId), ['yibing_touxi', 'mouding_houfa'])
	assert.equal(resolveReportTraitDisplaySide(hit, hit.traits?.[0]), 'reinforcement')
	assert.equal(resolveReportTraitDisplaySide(hit, hit.traits?.[1]), 'reinforcement')
	assert.equal(formatTraitOutcomeDetail('preBattleAffected', hit.traits?.[0].detail.preBattleAffected, options), '战前真实伤亡: 蜀步兵 +35')
	assert.equal(formatTraitOutcomeDetail('infantryDefenseModifiedUnits', hit.traits?.[1].detail.infantryDefenseModifiedUnits), '实际步防修正: weiInfantry +4')
	assert.equal(hasTraitEntries(miss), false)
})

test('关羽援军随机战前特性命中或未命中时保持准确兵力和方向', () => {
	const cases = [
		{
			generalId: 'guanyu', generalName: '关羽', helperFaction: 'shu', helperUnit: 'shuInfantry', traitId: 'shuiyan_qijun', traitName: '水淹七军', bonusId: 'wusheng_pojun', bonusName: '武圣破军',
			attackerFaction: 'wei', attackerUnit: 'weiInfantry', attackerUnitName: '魏步兵', hitPower: 650, hitAttackerLosses: 77, hitDefendingLosses: 35, hitReinforcementLosses: 35,
			detail: { effectRate: 0.35, maxAffectedRate: 0.35, preBattleAffected: { weiInfantry: 35 }, triggerChance: 1 },
		},
	] as const
	const options = { units: { shu: { shuInfantry: { name: '蜀步兵' } }, wei: { weiInfantry: { name: '魏步兵' } } } }

	for (const current of cases) {
		const buildReport = (triggered: boolean) => ({
			id: `${current.generalId}-reinforcement-${triggered ? 'hit' : 'miss'}`, playerId: 'attacker', ownerPlayerId: 'attacker', viewType: 'attack', sourceType: 'player_city',
			battleType: 'plunder', type: 'attack', result: triggered ? 'defender_victory' : 'draw', defenderRevealed: true, rewards: {}, read: true, createdAt: '2026-07-20T23:00:00Z',
			pvpReinforcements: [{
				reinforcementId: `rein_${current.generalId}`, fromPlayerId: `helper_${current.generalId}`, fromPlayerName: `${current.generalName}援军`, faction: current.helperFaction,
				troops: { [current.helperUnit]: 99 }, generalExpGained: triggered ? current.hitAttackerLosses : 50,
				generals: [{
					id: current.generalId, name: current.generalName, level: 1,
					traits: [{ traitId: current.traitId, name: current.traitName }, { traitId: current.bonusId, name: current.bonusName, allowedSides: ['attacker'] }],
				}],
			}],
			pvpReinforcementLosses: { [`rein_${current.generalId}`]: { [current.helperUnit]: triggered ? current.hitReinforcementLosses : 49 } },
			detail: {
				id: `${current.generalId}-reinforcement-${triggered ? 'hit' : 'miss'}`, sourceType: 'player_city', viewType: 'attack', battleType: 'plunder',
				result: triggered ? 'defender_victory' : 'draw', winnerSide: triggered ? 'defender' : 'none',
				primarySide: {
					role: 'attacker', faction: current.attackerFaction, power: triggered ? current.hitPower : 1000, generals: [{ id: 'attacker_general', name: '进攻将领', level: 1 }],
					units: [{ unitType: current.attackerUnit, unitName: current.attackerUnitName, amountBefore: 100, dispatched: 100, lost: triggered ? current.hitAttackerLosses : 50, survived: triggered ? 100 - current.hitAttackerLosses : 50 }],
				},
				secondarySide: {
					role: 'defender', faction: 'wu', power: 1000, generals: [{ id: 'sunquan', name: '孙权', level: 1 }],
					units: [{ unitType: 'wuInfantry', unitName: '吴步兵', amountBefore: 1, dispatched: 1, lost: triggered ? 0 : 1, survived: triggered ? 1 : 0 }],
				},
				rewards: { generalExp: triggered ? current.hitDefendingLosses : 50, resources: {} },
				traits: triggered ? [{
					traitId: current.traitId, traitName: current.traitName, ownerSide: 'reinforcement', ownerRole: 'reinforcement', ownerPlayerId: `helper_${current.generalId}`, generalId: current.generalId,
					detail: current.detail,
				}] : [],
			},
		})
		const hit = normalizeBattleReportDetail(buildReport(true))
		const miss = normalizeBattleReportDetail(buildReport(false))
		const hitPvp = hit.extra?.pvp && typeof hit.extra.pvp === 'object' ? hit.extra.pvp : undefined
		const missPvp = miss.extra?.pvp && typeof miss.extra.pvp === 'object' ? miss.extra.pvp : undefined

		assert.deepEqual(hit.primarySide.units[0], { unitType: current.attackerUnit, unitName: current.attackerUnitName, amountBefore: 100, dispatched: 100, lost: current.hitAttackerLosses, survived: 100 - current.hitAttackerLosses })
		assert.deepEqual(miss.primarySide.units[0], { unitType: current.attackerUnit, unitName: current.attackerUnitName, amountBefore: 100, dispatched: 100, lost: 50, survived: 50 })
		assert.deepEqual(hit.secondarySide?.units[0], { unitType: 'wuInfantry', unitName: '吴步兵', amountBefore: 1, dispatched: 1, lost: 0, survived: 1 })
		assert.deepEqual(miss.secondarySide?.units[0], { unitType: 'wuInfantry', unitName: '吴步兵', amountBefore: 1, dispatched: 1, lost: 1, survived: 0 })
		assert.deepEqual(hitPvp?.reinforcementLosses, { [`rein_${current.generalId}`]: { [current.helperUnit]: current.hitReinforcementLosses } })
		assert.deepEqual(missPvp?.reinforcementLosses, { [`rein_${current.generalId}`]: { [current.helperUnit]: 49 } })
		for (const pvp of [hitPvp, missPvp]) {
			assert.deepEqual(pvp?.reinforcements[0]?.generals?.[0]?.traits?.map((trait) => trait.traitId), [current.traitId, current.bonusId])
		}
		assert.equal(hitPvp?.reinforcements[0]?.generalExpGained, current.hitAttackerLosses)
		assert.equal(missPvp?.reinforcements[0]?.generalExpGained, 50)
		assert.deepEqual(hit.traits?.map((trait) => trait.traitId), [current.traitId])
		assert.equal(resolveReportTraitDisplaySide(hit, hit.traits?.[0]), 'reinforcement')
		assert.equal(hasTraitEntries(miss), false)
		assert.equal([...(hit.traits ?? []), ...(miss.traits ?? [])].some((trait) => trait.traitId === current.bonusId), false)
		assert.equal(formatTraitOutcomeDetail('preBattleAffected', hit.traits?.[0].detail.preBattleAffected, options), '战前真实伤亡: 魏步兵 +35')
	}
})

test('张飞援军震慑命中或未命中时区分临时压制与真实阵亡', () => {
	const buildReport = (triggered: boolean) => ({
		id: `zhangfei-reinforcement-${triggered ? 'hit' : 'miss'}`, playerId: 'attacker', ownerPlayerId: 'attacker', viewType: 'attack', sourceType: 'player_city',
		battleType: 'plunder', type: 'attack', result: triggered ? 'defender_victory' : 'draw', defenderRevealed: true, rewards: {}, read: true, createdAt: '2026-07-20T19:00:00Z',
		pvpReinforcements: [{
			reinforcementId: 'rein_zhangfei', fromPlayerId: 'helper_zhangfei', fromPlayerName: '张飞援军', faction: 'shu',
			troops: { shuInfantry: 99 }, generalExpGained: triggered ? 36 : 50,
			generals: [{
				id: 'zhangfei', name: '张飞', level: 1,
				traits: [{ traitId: 'zhenhe_quanjun', name: '震慑全军' }, { traitId: 'wanren_nuhou', name: '万人怒吼', allowedSides: ['attacker'] }],
			}],
		}],
		pvpReinforcementLosses: { rein_zhangfei: { shuInfantry: triggered ? 27 : 49 } },
		detail: {
			id: `zhangfei-reinforcement-${triggered ? 'hit' : 'miss'}`, sourceType: 'player_city', viewType: 'attack', battleType: 'plunder',
			result: triggered ? 'defender_victory' : 'draw', winnerSide: triggered ? 'defender' : 'none',
			primarySide: {
				role: 'attacker', faction: 'wei', power: triggered ? 500 : 1000, generals: [{ id: 'caocao', name: '曹操', level: 1 }],
				units: [{ unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 100, dispatched: 100, lost: triggered ? 36 : 50, survived: triggered ? 64 : 50 }],
			},
			secondarySide: {
				role: 'defender', faction: 'wu', power: 1000, generals: [{ id: 'sunquan', name: '孙权', level: 1 }],
				units: [{ unitType: 'wuInfantry', unitName: '吴步兵', amountBefore: 1, dispatched: 1, lost: triggered ? 0 : 1, survived: triggered ? 1 : 0 }],
			},
			rewards: { generalExp: triggered ? 27 : 50, resources: {} },
			traits: triggered ? [{
				traitId: 'zhenhe_quanjun', traitName: '震慑全军', ownerSide: 'reinforcement', ownerRole: 'reinforcement', ownerPlayerId: 'helper_zhangfei', generalId: 'zhangfei',
				detail: { effectRate: 0.5, maxAffectedRate: 0.5, suppressedUnits: { weiInfantry: 50 }, triggerChance: 1 },
			}] : [],
		},
	})
	const hit = normalizeBattleReportDetail(buildReport(true))
	const miss = normalizeBattleReportDetail(buildReport(false))
	const hitPvp = hit.extra?.pvp && typeof hit.extra.pvp === 'object' ? hit.extra.pvp : undefined
	const missPvp = miss.extra?.pvp && typeof miss.extra.pvp === 'object' ? miss.extra.pvp : undefined
	const options = { faction: 'wei', units: { wei: { weiInfantry: { name: '魏步兵' } } } }

	assert.deepEqual(hit.primarySide.units[0], { unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 100, dispatched: 100, lost: 36, survived: 64 })
	assert.deepEqual(miss.primarySide.units[0], { unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 100, dispatched: 100, lost: 50, survived: 50 })
	assert.deepEqual(hit.secondarySide?.units[0], { unitType: 'wuInfantry', unitName: '吴步兵', amountBefore: 1, dispatched: 1, lost: 0, survived: 1 })
	assert.deepEqual(miss.secondarySide?.units[0], { unitType: 'wuInfantry', unitName: '吴步兵', amountBefore: 1, dispatched: 1, lost: 1, survived: 0 })
	assert.deepEqual(hitPvp?.reinforcementLosses, { rein_zhangfei: { shuInfantry: 27 } })
	assert.deepEqual(missPvp?.reinforcementLosses, { rein_zhangfei: { shuInfantry: 49 } })
	assert.equal(hitPvp?.reinforcements[0]?.generalExpGained, 36)
	assert.equal(missPvp?.reinforcements[0]?.generalExpGained, 50)
	for (const pvp of [hitPvp, missPvp]) {
		assert.deepEqual(pvp?.reinforcements[0]?.generals?.[0]?.traits?.map((trait) => trait.traitId), ['zhenhe_quanjun', 'wanren_nuhou'])
	}
	assert.deepEqual(hit.traits?.map((trait) => trait.traitId), ['zhenhe_quanjun'])
	assert.equal(resolveReportTraitDisplaySide(hit, hit.traits?.[0]), 'reinforcement')
	assert.equal(formatTraitOutcomeDetail('suppressedUnits', hit.traits?.[0].detail.suppressedUnits, options), '本场压制兵力: 魏步兵 +50')
	assert.equal(hasTraitEntries(miss), false)
	assert.equal([...(hit.traits ?? []), ...(miss.traits ?? [])].some((trait) => trait.traitId === 'wanren_nuhou'), false)
})

test('黄盖援军苦肉命中或未命中时保持双方后续特性和精确兵损', () => {
	const buildReport = (triggered: boolean) => ({
		id: `huanggai-reinforcement-${triggered ? 'hit' : 'miss'}`, playerId: 'attacker', ownerPlayerId: 'attacker', viewType: 'attack', sourceType: 'player_city',
		battleType: 'plunder', type: 'attack', result: 'draw', defenderRevealed: true, rewards: {}, read: true, createdAt: '2026-07-20T20:00:00Z',
		pvpReinforcements: [{
			reinforcementId: 'rein_huanggai', fromPlayerId: 'helper_huanggai', fromPlayerName: '黄盖援军', faction: 'wu',
			troops: { wuInfantry: 99 }, generalExpGained: 60,
			generals: [{
				id: 'huanggai', name: '黄盖', level: 1,
				traits: [{ traitId: 'kurouji', name: '苦肉计' }, { traitId: 'kurou_fanji', name: '苦肉反击' }],
			}],
		}],
		pvpReinforcementLosses: { rein_huanggai: { wuInfantry: triggered ? 49 : 59 } },
		detail: {
			id: `huanggai-reinforcement-${triggered ? 'hit' : 'miss'}`, sourceType: 'player_city', viewType: 'attack', battleType: 'plunder', result: 'draw', winnerSide: 'none',
			primarySide: {
				role: 'attacker', faction: 'shu', power: 1000,
				generals: [{ id: 'huangzhong', name: '黄忠', level: 1, traits: [{ traitId: 'laodang_yizhuang', name: '老当益壮' }] }],
				units: [{ unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 100, dispatched: 100, lost: 60, survived: 40 }],
			},
			secondarySide: {
				role: 'defender', faction: 'wu', power: 1000, generals: [{ id: 'sunquan', name: '孙权', level: 1 }],
				units: [{ unitType: 'wuInfantry', unitName: '吴步兵', amountBefore: 1, dispatched: 1, lost: 1, survived: 0 }],
			},
			rewards: { generalExp: triggered ? 50 : 60, resources: {} },
			traits: triggered ? [
				{
					traitId: 'kurouji', traitName: '苦肉计', ownerSide: 'reinforcement', ownerRole: 'reinforcement', ownerPlayerId: 'helper_huanggai', generalId: 'huanggai',
					detail: { disableTraitCount: 1, disabledTraitCount: 1, triggerChance: 1 },
				},
				{
					traitId: 'kurou_fanji', traitName: '苦肉反击', ownerSide: 'reinforcement', ownerRole: 'reinforcement', ownerPlayerId: 'helper_huanggai', generalId: 'huanggai',
					detail: { effectRate: 0.1, extraLosses: { shuInfantry: 10 } },
				},
			] : [
				{
					traitId: 'laodang_yizhuang', traitName: '老当益壮', ownerSide: 'primary', ownerRole: 'attacker', ownerPlayerId: 'attacker', generalId: 'huangzhong',
					detail: { effectRate: 0.1, extraLosses: { wuInfantry: 10 } },
				},
				{
					traitId: 'kurou_fanji', traitName: '苦肉反击', ownerSide: 'reinforcement', ownerRole: 'reinforcement', ownerPlayerId: 'helper_huanggai', generalId: 'huanggai',
					detail: { effectRate: 0.1, extraLosses: { shuInfantry: 10 } },
				},
			],
		},
	})
	const hit = normalizeBattleReportDetail(buildReport(true))
	const miss = normalizeBattleReportDetail(buildReport(false))
	const hitPvp = hit.extra?.pvp && typeof hit.extra.pvp === 'object' ? hit.extra.pvp : undefined
	const missPvp = miss.extra?.pvp && typeof miss.extra.pvp === 'object' ? miss.extra.pvp : undefined
	const options = { faction: 'shu', units: { shu: { shuInfantry: { name: '蜀步兵' } }, wu: { wuInfantry: { name: '吴步兵' } } } }

	for (const current of [hit, miss]) {
		assert.deepEqual(current.primarySide.units[0], { unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 100, dispatched: 100, lost: 60, survived: 40 })
		assert.deepEqual(current.secondarySide?.units[0], { unitType: 'wuInfantry', unitName: '吴步兵', amountBefore: 1, dispatched: 1, lost: 1, survived: 0 })
	}
	assert.deepEqual(hitPvp?.reinforcementLosses, { rein_huanggai: { wuInfantry: 49 } })
	assert.deepEqual(missPvp?.reinforcementLosses, { rein_huanggai: { wuInfantry: 59 } })
	for (const pvp of [hitPvp, missPvp]) {
		assert.equal(pvp?.reinforcements[0]?.generalExpGained, 60)
		assert.deepEqual(pvp?.reinforcements[0]?.generals?.[0]?.traits?.map((trait) => trait.traitId), ['kurouji', 'kurou_fanji'])
	}
	assert.deepEqual(hit.traits?.map((trait) => trait.traitId), ['kurouji', 'kurou_fanji'])
	assert.deepEqual(miss.traits?.map((trait) => trait.traitId), ['laodang_yizhuang', 'kurou_fanji'])
	assert.deepEqual(hit.traits?.map((trait) => resolveReportTraitDisplaySide(hit, trait)), ['reinforcement', 'reinforcement'])
	assert.deepEqual(miss.traits?.map((trait) => resolveReportTraitDisplaySide(miss, trait)), ['primary', 'reinforcement'])
	assert.equal(formatTraitOutcomeDetail('disabledTraitCount', hit.traits?.[0].detail.disabledTraitCount, options), '实际压制特性数: 1')
	assert.equal(formatTraitOutcomeDetail('extraLosses', hit.traits?.[1].detail.extraLosses, options), '追加损失: 蜀步兵 +10')
	assert.equal(formatTraitOutcomeDetail('extraLosses', miss.traits?.[0].detail.extraLosses, options), '追加损失: 吴步兵 +10')
	assert.equal(formatTraitOutcomeDetail('extraLosses', miss.traits?.[1].detail.extraLosses, options), '追加损失: 蜀步兵 +10')
	assert.equal(hit.traits?.some((trait) => trait.traitId === 'laodang_yizhuang'), false)
	assert.equal(miss.traits?.some((trait) => trait.traitId === 'kurouji'), false)
})

test('马超援军西凉突击只追加骑兵损失且被动武力不进入时间线', () => {
	const buildReport = (triggered: boolean) => ({
		id: `machao-reinforcement-${triggered ? 'hit' : 'miss'}`, playerId: 'attacker', ownerPlayerId: 'attacker', viewType: 'attack', sourceType: 'player_city',
		battleType: 'plunder', type: 'attack', result: 'attacker_victory', defenderRevealed: true, rewards: {}, read: true, createdAt: '2026-07-20T21:00:00Z',
		pvpReinforcements: [{
			reinforcementId: 'rein_machao', fromPlayerId: 'helper_machao', fromPlayerName: '马超援军', faction: 'shu',
			troops: { shuInfantry: 99 }, generalExpGained: triggered ? 71 : 59,
			generals: [{
				id: 'machao', name: '马超', level: 1, stats: { force: 0 }, effectiveStats: { force: 20 }, buffs: { attackBonus: 0.4 },
				traits: [{ traitId: 'xiliang_tuji', name: '西凉突击' }, { traitId: 'tianshen_xiafan', name: '天神下凡' }],
			}],
		}],
		pvpReinforcementLosses: { rein_machao: { shuInfantry: 60 } },
		detail: {
			id: `machao-reinforcement-${triggered ? 'hit' : 'miss'}`, sourceType: 'player_city', viewType: 'attack', battleType: 'plunder', result: 'attacker_victory', winnerSide: 'attacker',
			primarySide: {
				role: 'attacker', faction: 'wei', power: 1200, generals: [{ id: 'caocao', name: '曹操', level: 1 }],
				units: [
					{ unitType: 'weiCavalry', unitName: '魏骑兵', amountBefore: 50, dispatched: 50, lost: triggered ? 26 : 20, survived: triggered ? 24 : 30 },
					{ unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 50, dispatched: 50, lost: 19, survived: 31 },
				],
			},
			secondarySide: {
				role: 'defender', faction: 'wu', power: 883.3333333333334, generals: [{ id: 'sunquan', name: '孙权', level: 1 }],
				units: [{ unitType: 'wuInfantry', unitName: '吴步兵', amountBefore: 1, dispatched: 1, lost: 0, survived: 1 }],
			},
			rewards: { generalExp: 60, resources: {} },
			traits: triggered ? [{
				traitId: 'xiliang_tuji', traitName: '西凉突击', ownerSide: 'reinforcement', ownerRole: 'reinforcement', ownerPlayerId: 'helper_machao', generalId: 'machao',
				detail: { effectRate: 0.12, targetExtraLosses: { weiCavalry: 6 }, triggerChance: 1 },
			}] : [],
		},
	})
	const hit = normalizeBattleReportDetail(buildReport(true))
	const miss = normalizeBattleReportDetail(buildReport(false))
	const hitPvp = hit.extra?.pvp && typeof hit.extra.pvp === 'object' ? hit.extra.pvp : undefined
	const missPvp = miss.extra?.pvp && typeof miss.extra.pvp === 'object' ? miss.extra.pvp : undefined
	const options = { faction: 'wei', units: { wei: { weiCavalry: { name: '魏骑兵' }, weiInfantry: { name: '魏步兵' } } } }

	assert.deepEqual(hit.primarySide.units.map((unit) => [unit.unitType, unit.lost, unit.survived]), [['weiCavalry', 26, 24], ['weiInfantry', 19, 31]])
	assert.deepEqual(miss.primarySide.units.map((unit) => [unit.unitType, unit.lost, unit.survived]), [['weiCavalry', 20, 30], ['weiInfantry', 19, 31]])
	for (const current of [hit, miss]) {
		assert.deepEqual(current.secondarySide?.units[0], { unitType: 'wuInfantry', unitName: '吴步兵', amountBefore: 1, dispatched: 1, lost: 0, survived: 1 })
	}
	for (const pvp of [hitPvp, missPvp]) {
		assert.deepEqual(pvp?.reinforcementLosses, { rein_machao: { shuInfantry: 60 } })
		const snapshot = pvp?.reinforcements[0]?.generals?.[0]
		assert.equal(snapshot?.id, 'machao')
		assert.deepEqual(snapshot?.stats, { force: 0 })
		assert.deepEqual(snapshot?.effectiveStats, { force: 20 })
		assert.deepEqual(snapshot?.buffs, { attackBonus: 0.4 })
		assert.deepEqual(snapshot?.traits?.map((trait) => trait.traitId), ['xiliang_tuji', 'tianshen_xiafan'])
	}
	assert.equal(hitPvp?.reinforcements[0]?.generalExpGained, 71)
	assert.equal(missPvp?.reinforcements[0]?.generalExpGained, 59)
	assert.deepEqual(hit.traits?.map((trait) => trait.traitId), ['xiliang_tuji'])
	assert.equal(resolveReportTraitDisplaySide(hit, hit.traits?.[0]), 'reinforcement')
	assert.equal(formatTraitOutcomeDetail('targetExtraLosses', hit.traits?.[0].detail.targetExtraLosses, options), '目标兵种追加损失: 魏骑兵 +6')
	assert.equal(hasTraitEntries(miss), false)
	assert.equal([...(hit.traits ?? []), ...(miss.traits ?? [])].some((trait) => trait.traitId === 'tianshen_xiafan'), false)
})

test('诸葛亮援军战前困兵与全体封禁保持真实归属并对齐最终兵力', () => {
	const report = {
		id: 'zhugeliang-reinforcement-active', playerId: 'attacker', ownerPlayerId: 'attacker', viewType: 'attack', sourceType: 'player_city',
		battleType: 'plunder', type: 'attack', result: 'defender_victory', defenderRevealed: true, rewards: {}, read: true, createdAt: '2026-07-20T22:00:00Z',
		pvpReinforcements: [{
			reinforcementId: 'rein_zhugeliang', fromPlayerId: 'helper_zhugeliang', fromPlayerName: '诸葛亮援军', faction: 'shu',
			troops: { shuInfantry: 99 }, generalExpGained: 45,
			generals: [{
				id: 'zhugeliang', name: '诸葛亮', level: 1,
				traits: [{ traitId: 'qimen_dunjia', name: '奇门遁甲' }, { traitId: 'wolong_mouzhi', name: '卧龙奇谋' }],
			}],
		}],
		pvpReinforcementLosses: { rein_zhugeliang: { shuInfantry: 39 } },
		detail: {
			id: 'zhugeliang-reinforcement-active', sourceType: 'player_city', viewType: 'attack', battleType: 'plunder', result: 'defender_victory', winnerSide: 'defender',
			primarySide: {
				role: 'attacker', faction: 'shu', power: 750, generals: [{ id: 'huangzhong', name: '黄忠', level: 1 }],
				units: [{ unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 100, dispatched: 100, lost: 45, survived: 55 }],
			},
			secondarySide: {
				role: 'defender', faction: 'wu', power: 1000, generals: [{ id: 'sunquan', name: '孙权', level: 1 }],
				units: [{ unitType: 'wuInfantry', unitName: '吴步兵', amountBefore: 1, dispatched: 1, lost: 0, survived: 1 }],
			},
			rewards: { generalExp: 39, resources: {} },
			traits: [
				{
					traitId: 'qimen_dunjia', traitName: '奇门遁甲', ownerSide: 'reinforcement', ownerRole: 'reinforcement', ownerPlayerId: 'helper_zhugeliang', generalId: 'zhugeliang',
					detail: { effectRate: 0.25, suppressedUnits: { shuInfantry: 25 }, triggerChance: 1 },
				},
				{
					traitId: 'wolong_mouzhi', traitName: '卧龙奇谋', ownerSide: 'reinforcement', ownerRole: 'reinforcement', ownerPlayerId: 'helper_zhugeliang', generalId: 'zhugeliang',
					detail: { disabledGeneralCount: 1, disabledTraitCount: 1, triggerChance: 1 },
				},
			],
		},
	}
	const detail = normalizeBattleReportDetail(report)
	const pvp = detail.extra?.pvp && typeof detail.extra.pvp === 'object' ? detail.extra.pvp : undefined
	const options = { faction: 'shu', units: { shu: { shuInfantry: { name: '蜀步兵' } } } }

	assert.equal(detail.primarySide.power, 750)
	assert.deepEqual(detail.primarySide.units[0], { unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 100, dispatched: 100, lost: 45, survived: 55 })
	assert.deepEqual(detail.secondarySide?.units[0], { unitType: 'wuInfantry', unitName: '吴步兵', amountBefore: 1, dispatched: 1, lost: 0, survived: 1 })
	assert.deepEqual(pvp?.reinforcementLosses, { rein_zhugeliang: { shuInfantry: 39 } })
	assert.equal(pvp?.reinforcements[0]?.generalExpGained, 45)
	assert.deepEqual(pvp?.reinforcements[0]?.generals?.[0]?.traits?.map((trait) => trait.traitId), ['qimen_dunjia', 'wolong_mouzhi'])
	assert.deepEqual(detail.traits?.map((trait) => trait.traitId), ['qimen_dunjia', 'wolong_mouzhi'])
	assert.deepEqual(detail.traits?.map((trait) => resolveReportTraitDisplaySide(detail, trait)), ['reinforcement', 'reinforcement'])
	assert.equal(formatTraitOutcomeDetail('suppressedUnits', detail.traits?.[0].detail.suppressedUnits, options), '本场压制兵力: 蜀步兵 +25')
	assert.equal(formatTraitOutcomeDetail('disabledGeneralCount', detail.traits?.[1].detail.disabledGeneralCount, options), '封禁将领数: 1')
	assert.equal(formatTraitOutcomeDetail('disabledTraitCount', detail.traits?.[1].detail.disabledTraitCount, options), '实际压制特性数: 1')
	assert.equal(detail.traits?.some((trait) => trait.traitId === 'laodang_yizhuang'), false)
})

test('孙策固定加攻与司马懿全军加防分别归属攻守侧', () => {
	const detail = {
		sourceType: 'player_city', viewType: 'attack', battleType: 'attack',
		primarySide: {
			role: 'attacker', power: 16000,
			units: [{ unitType: 'overlordRider', unitName: '霸王骑', amountBefore: 200, dispatched: 200, lost: 90, survived: 110 }],
		},
		secondarySide: {
			role: 'defender', power: 14000,
			units: [{ unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 1000, dispatched: 1000, lost: 1000, survived: 0 }],
		},
		traits: [
			{
				traitId: 'xiaobawang_tieqi', traitName: '小霸王', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'sunce',
				detail: { unitAttackFlat: 50, attackModifiedUnits: { overlordRider: 50 } },
			},
			{
				traitId: 'mouding_houfa', traitName: '谋定后发', ownerSide: 'secondary', ownerRole: 'defender', generalId: 'simayi',
				detail: { defenseBonusRate: 0.35, infantryDefenseModifiedUnits: { weiInfantry: 4 }, cavalryDefenseModifiedUnits: { weiInfantry: 3 }, triggerChance: 1 },
			},
		],
	}
	const unitOptions = { faction: 'wu', units: { wu: { overlordRider: { name: '霸王骑' } }, wei: { weiInfantry: { name: '魏步兵' } } } }
	assert.deepEqual(detail.traits.map((trait) => trait.traitId), ['xiaobawang_tieqi', 'mouding_houfa'])
	assert.deepEqual(detail.traits.map((trait) => resolveReportTraitDisplaySide(detail, trait)), ['primary', 'secondary'])
	assert.deepEqual(detail.primarySide, { role: 'attacker', power: 16000, units: [{ unitType: 'overlordRider', unitName: '霸王骑', amountBefore: 200, dispatched: 200, lost: 90, survived: 110 }] })
	assert.deepEqual(detail.secondarySide, { role: 'defender', power: 14000, units: [{ unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 1000, dispatched: 1000, lost: 1000, survived: 0 }] })
	assert.equal(formatTraitOutcomeDetail('unitAttackFlat', detail.traits[0].detail.unitAttackFlat, unitOptions), '设计单位攻击增加: 50')
	assert.equal(formatTraitOutcomeDetail('attackModifiedUnits', detail.traits[0].detail.attackModifiedUnits, unitOptions), '实际攻击修正: 霸王骑 +50')
	assert.equal(formatTraitOutcomeDetail('defenseBonusRate', detail.traits[1].detail.defenseBonusRate, unitOptions), '设计防御加成: 35%')
	assert.equal(formatTraitOutcomeDetail('infantryDefenseModifiedUnits', detail.traits[1].detail.infantryDefenseModifiedUnits, unitOptions), '实际步防修正: 魏步兵 +4')
})

test('NPC 和黄巾战斗结算后四项追加伤害展示后端实际增量而不是配置值', () => {
	const options = { faction: 'wu', units: { wu: { shadowGuard: { name: '影卫' } } } }
	for (const [traitId, detailKey, amount] of [
		['laodang_yizhuang', 'extraLosses', 100],
		['huoshao_lianying', 'targetExtraLosses', 963],
		['lianying_zengshang', 'targetExtraLosses', 100],
		['kurou_fanji', 'extraLosses', 100],
	] as const) {
		assert.equal(getTraitMeta(traitId).trigger, '战斗结算后')
		assert.match(formatTraitOutcomeDetail(detailKey, { shadowGuard: amount }, options), new RegExp(`影卫 \\+${amount}`))
	}
	assert.equal(formatTraitOutcomeDetail('extraLosses', { shadowGuard: 100 }, options), '追加损失: 影卫 +100')
	assert.equal(formatTraitOutcomeDetail('targetExtraLosses', { shadowGuard: 963 }, options), '目标兵种追加损失: 影卫 +963')
	assert.equal(formatTraitOutcomeDetail('effectRate', 0.1), '设计效果比例: 10%')
	assert.equal(formatTraitOutcomeDetail('damagePercent', 0.25), '设计伤害比例: 25%')
	assert.match(getTraitMeta('huogong').description, /主动进攻/)
	assert.match(getTraitMeta('xiaobawang_zhuiji').description, /掠夺战胜利后/)
})

test('火烧联营明确按原始人数追加并使目标步兵最终全灭', () => {
	const trait = getTraitMeta('huoshao_lianying')
	assert.match(trait.description, /35% 概率/)
	assert.match(trait.description, /步兵战前人数追加 100%/)
	assert.match(trait.description, /最终全灭/)
})

test('陆逊双追加伤害只采用后端本场实际触发结果', () => {
	const generalSnapshot = {
		id: 'luxun', name: '陆逊',
		traits: [{ traitId: 'huoshao_lianying', name: '火烧联营' }, { traitId: 'lianying_zengshang', name: '连营增伤' }],
	}
	const bonusOnly = {
		primarySide: { generals: [generalSnapshot] },
		traits: [{
			traitId: 'lianying_zengshang', traitName: '连营增伤', ownerSide: 'primary', generalId: 'luxun',
			detail: { effectRate: 0.1, targetExtraLosses: { greedyWolf: 100 }, triggerChance: 1 },
		}],
	}
	assert.equal(hasTraitEntries(bonusOnly), true)
	assert.deepEqual(bonusOnly.traits.map((trait) => trait.traitId), ['lianying_zengshang'])
	assert.equal(formatTraitOutcomeDetail('targetExtraLosses', bonusOnly.traits[0].detail.targetExtraLosses), '目标兵种追加损失: 贪狼营 +100')

	const fireCapped = {
		primarySide: { generals: [generalSnapshot] },
		traits: [{
			traitId: 'huoshao_lianying', traitName: '火烧联营', ownerSide: 'primary', generalId: 'luxun',
			detail: { effectRate: 1, targetExtraLosses: { greedyWolf: 436 }, triggerChance: 1 },
		}],
	}
	assert.equal(hasTraitEntries(fireCapped), true)
	assert.deepEqual(fireCapped.traits.map((trait) => trait.traitId), ['huoshao_lianying'])
	assert.equal(formatTraitOutcomeDetail('targetExtraLosses', fireCapped.traits[0].detail.targetExtraLosses), '目标兵种追加损失: 贪狼营 +436')
	assert.equal(fireCapped.traits.some((trait) => trait.traitId === 'lianying_zengshang'), false)
})

test('陆逊火烧未命中时连营增伤仍按真实兵力追加损失', () => {
	const detail = {
		sourceType: 'player_city', viewType: 'attack', battleType: 'plunder',
		primarySide: {
			role: 'attacker', power: 10000,
			generals: [{
				id: 'luxun', name: '陆逊', level: 1,
				traits: [{ traitId: 'huoshao_lianying', name: '火烧联营' }, { traitId: 'lianying_zengshang', name: '连营增伤' }],
			}],
			units: [{ unitType: 'wuInfantry', unitName: '吴步兵', amountBefore: 1000, dispatched: 1000, lost: 500, survived: 500 }],
		},
		secondarySide: {
			role: 'defender', power: 10000,
			units: [{ unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 1000, dispatched: 1000, lost: 600, survived: 400 }],
		},
		rewards: { generalExp: 600, resources: {} },
		traits: [{
			traitId: 'lianying_zengshang', traitName: '连营增伤', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'luxun',
			detail: { effectRate: 0.1, targetExtraLosses: { weiInfantry: 100 } },
		}],
	}
	const unitOptions = { faction: 'wu', units: { wei: { weiInfantry: { name: '魏步兵' } } } }
	assert.deepEqual(detail.primarySide.generals[0].traits.map((trait) => trait.traitId), ['huoshao_lianying', 'lianying_zengshang'])
	assert.deepEqual(detail.traits.map((trait) => trait.traitId), ['lianying_zengshang'])
	assert.equal(detail.traits.some((trait) => trait.traitId === 'huoshao_lianying'), false)
	assert.deepEqual(detail.primarySide.units[0], { unitType: 'wuInfantry', unitName: '吴步兵', amountBefore: 1000, dispatched: 1000, lost: 500, survived: 500 })
	assert.deepEqual(detail.secondarySide.units[0], { unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 1000, dispatched: 1000, lost: 600, survived: 400 })
	assert.equal(formatTraitOutcomeDetail('effectRate', detail.traits[0].detail.effectRate, unitOptions), '设计效果比例: 10%')
	assert.equal(formatTraitOutcomeDetail('targetExtraLosses', detail.traits[0].detail.targetExtraLosses, unitOptions), '目标兵种追加损失: 魏步兵 +100')
})

test('防守陆逊火烧未命中时连营增伤只追加来袭步兵损失', () => {
	const detail = {
		sourceType: 'player_city', viewType: 'defense', battleType: 'plunder', ownerSide: 'defender',
		primarySide: {
			role: 'attacker', power: 10000,
			units: [{ unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 1000, dispatched: 1000, lost: 600, survived: 400 }],
		},
		secondarySide: {
			role: 'defender', power: 10000,
			generals: [{
				id: 'luxun', name: '陆逊', level: 1,
				traits: [{ traitId: 'huoshao_lianying', name: '火烧联营' }, { traitId: 'lianying_zengshang', name: '连营增伤' }],
			}],
			units: [{ unitType: 'wuInfantry', unitName: '吴步兵', amountBefore: 1000, dispatched: 1000, lost: 500, survived: 500 }],
		},
		rewards: { generalExp: 600, resources: {} },
		traits: [{
			traitId: 'lianying_zengshang', traitName: '连营增伤', ownerSide: 'secondary', ownerRole: 'defender', generalId: 'luxun',
			detail: { effectRate: 0.1, targetExtraLosses: { shuInfantry: 100 } },
		}],
	}
	const unitOptions = { faction: 'wu', units: { shu: { shuInfantry: { name: '蜀步兵' } }, wu: { wuInfantry: { name: '吴步兵' } } } }
	assert.deepEqual(detail.secondarySide.generals[0].traits.map((trait) => trait.traitId), ['huoshao_lianying', 'lianying_zengshang'])
	assert.deepEqual(detail.traits.map((trait) => trait.traitId), ['lianying_zengshang'])
	assert.equal(detail.traits.some((trait) => trait.traitId === 'huoshao_lianying'), false)
	assert.deepEqual(detail.traits.map((trait) => resolveReportTraitDisplaySide(detail, trait)), ['secondary'])
	assert.deepEqual(detail.primarySide.units[0], { unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 1000, dispatched: 1000, lost: 600, survived: 400 })
	assert.deepEqual(detail.secondarySide.units[0], { unitType: 'wuInfantry', unitName: '吴步兵', amountBefore: 1000, dispatched: 1000, lost: 500, survived: 500 })
	assert.equal(formatTraitOutcomeDetail('effectRate', detail.traits[0].detail.effectRate, unitOptions), '设计效果比例: 10%')
	assert.equal(formatTraitOutcomeDetail('targetExtraLosses', detail.traits[0].detail.targetExtraLosses, unitOptions), '目标兵种追加损失: 蜀步兵 +100')
	assert.deepEqual(detail.rewards, { generalExp: 600, resources: {} })
})

test('西凉突击在 NPC 战报中只展示敌方骑兵实际追加损失', () => {
	const trait = getTraitMeta('xiliang_tuji')
	const options = { faction: 'wei', units: { wei: { weiCavalry: { name: '魏骑兵' } } } }
	assert.equal(trait.trigger, '战斗结算后')
  assert.match(trait.description, /进攻、守城或作为援军/)
	assert.match(trait.description, /敌方骑兵战前人数/)
	assert.equal(formatTraitOutcomeDetail('targetExtraLosses', { weiCavalry: 60 }, options), '目标兵种追加损失: 魏骑兵 +60')
})

test('卧龙全体封禁与苦肉单项压制使用各自准确语义', () => {
	const wolong = getTraitMeta('wolong_mouzhi')
	assert.equal(wolong.name, '卧龙奇谋')
	assert.equal(wolong.trigger, '进攻/防守/增援战斗前')
	assert.match(wolong.description, /60% 概率/)
	assert.match(wolong.description, /所有参战将领/)
	assert.match(wolong.description, /不影响永久被动/)
	assert.match(wolong.description, /双方都有诸葛亮.*均失效/)
	const kurou = getTraitMeta('kurouji')
	assert.equal(kurou.trigger, '战斗结算后')
	assert.match(kurou.description, /拦截敌方 1 项后续特性/)
	assert.match(kurou.description, /实际压制数为 0/)
	assert.match(kurou.description, /35%/)
	assert.equal(formatTraitOutcomeDetail('disableTraitCount', 1), '设计压制特性数: 1')
	assert.equal(formatTraitOutcomeDetail('disabledGeneralCount', 2), '封禁将领数: 2')
	assert.equal(formatTraitOutcomeDetail('disabledTraitCount', 1), '实际压制特性数: 1')
	assert.equal(formatTraitOutcomeDetail('disabledTraitCount', 0), '实际压制特性数: 0')
	assert.equal(formatTraitOutcomeDetail('status', '特性已失效'), '状态: 特性已失效')
	assert.equal(formatTraitOutcomeDetail('invalidReason', '双方均有诸葛亮'), '失效原因: 双方均有诸葛亮')
	assert.equal(formatTraitOutcomeDetail('disabledTraits', { disabledTraitCount: 1 }), '压制特性: 实际压制特性数 +1')
	const noTargetDetail = {
		sourceType: 'player_city', viewType: 'attack',
		traits: [
			{ traitId: 'kurouji', ownerSide: 'secondary', detail: { disableTraitCount: 1, disabledTraitCount: 0 } },
		],
	}
	assert.equal(hasTraitEntries(noTargetDetail), true)
	assert.deepEqual(noTargetDetail.traits.map((trait) => formatTraitOutcomeDetail('disabledTraitCount', trait.detail.disabledTraitCount)), [
		'实际压制特性数: 0',
	])
})

test('诸葛亮双特性按后端时间线区分临时压兵和战前全体封禁', () => {
	const detail = {
		sourceType: 'player_city', viewType: 'attack',
		traits: [
			{
				traitId: 'qimen_dunjia', traitName: '奇门遁甲', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'zhugeliang',
				detail: { effectRate: 0.25, suppressedUnits: { greedyWolf: 25 }, triggerChance: 1 },
			},
			{
				traitId: 'wolong_mouzhi', traitName: '卧龙奇谋', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'zhugeliang',
				detail: { disabledGeneralCount: 1, disabledTraitCount: 2, triggerChance: 1 },
			},
		],
	}
	assert.equal(hasTraitEntries(detail), true)
	assert.deepEqual(detail.traits.map((trait) => trait.traitId), ['qimen_dunjia', 'wolong_mouzhi'])
	assert.deepEqual(detail.traits.map((trait) => getTraitMeta(trait.traitId).trigger), ['进攻/防守/增援战斗前', '进攻/防守/增援战斗前'])
	assert.deepEqual(detail.traits.map((trait) => resolveReportTraitDisplaySide(detail, trait)), ['primary', 'primary'])
	assert.equal(formatTraitOutcomeDetail('suppressedUnits', detail.traits[0].detail.suppressedUnits), '本场压制兵力: 贪狼营 +25')
	assert.equal(formatTraitOutcomeDetails(detail.traits[1].detail), '封禁将领数: 1；实际压制特性数: 2；触发概率: 100%')
	assert.equal(detail.traits.some((trait) => trait.traitId === 'laodang_yizhuang'), false)
})

test('扫荡聚合战报显示特性真实触发场次', () => {
	assert.equal(formatTraitOutcomeDetail('triggerCount', 2), '触发场次: 2')
	const source = readFileSync(new URL('../src/pages/news/components/report-detail/ReportSweepContext.tsx', import.meta.url), 'utf8')
	assert.match(source, /已结算/)
	assert.match(source, /异常/)
	assert.doesNotMatch(source, /<span>成功/)
})

test('全部正式将领特性都有玩家可读中文元信息', () => {
	const traitIds = [...new Set(Object.values(GENERAL_TRAITS).flat())]
	assert.equal(traitIds.length, 50)
	for (const traitId of traitIds) {
		const meta = getTraitMeta(traitId)
		assert.notEqual(meta.name, traitId, `特性 ${traitId} 缺少中文名称`)
		assert.ok(meta.description.trim(), `特性 ${traitId} 缺少说明`)
	}
})

test('来源和视角标签使用不同颜色配置', () => {
  assert.match(REPORT_VIEW_CONFIG.attack.color, /red/)
  assert.match(REPORT_VIEW_CONFIG.defense.color, /blue/)
  assert.match(REPORT_VIEW_CONFIG.reinforcement.color, /green/)
  assert.match(REPORT_VIEW_CONFIG.scout.color, /yellow/)
  assert.match(REPORT_SOURCE_CONFIG.npc_city.color, /cyan/)
  assert.match(REPORT_SOURCE_CONFIG.player_city.color, /pink/)
  assert.match(REPORT_SOURCE_CONFIG.stronghold.color, /amber/)
  assert.equal(REPORT_SOURCE_CONFIG.dungeon.label, '副本')
  assert.match(REPORT_SOURCE_CONFIG.dungeon.color, /purple/)
})

test('轮回副本战报按后端时间线展示真实扣兵、攻击修正和仁主复活数', () => {
	const detail = {
		sourceType: 'dungeon', viewType: 'attack', battleType: 'dungeon_reincarnation_attack',
		traits: [
			{ traitId: 'shuiyan_qijun', traitName: '水淹七军', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'guanyu', detail: { effectRate: 0.35, maxAffectedRate: 0.35, preBattleAffected: { greedyWolf: 350 }, triggerChance: 1 } },
			{ traitId: 'meiren', traitName: '美人心计', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'zhenmi', detail: { attackBonusRate: 0.25, attackModifiedUnits: { huWei: 3 }, triggerChance: 0.5 } },
			{ traitId: 'renzhu_shouhu', traitName: '仁主守护', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'liubei', detail: { effectRate: 0.35, revivedUnits: { greedyWolf: 35 }, totalRevived: 35, triggerChance: 0.6 } },
		],
	}
	assert.deepEqual(detail.traits.map((trait) => resolveReportTraitDisplaySide(detail, trait)), ['primary', 'primary', 'primary'])
	assert.deepEqual(detail.traits.map((trait) => getTraitMeta(trait.traitId).trigger), ['战斗前', '主动进攻战斗前', '进攻/防守/增援战斗结束后'])
	assert.equal(formatTraitOutcomeDetail('preBattleAffected', detail.traits[0].detail.preBattleAffected), '战前真实伤亡: 贪狼营 +350')
	assert.equal(formatTraitOutcomeDetail('attackModifiedUnits', detail.traits[1].detail.attackModifiedUnits), '实际攻击修正: 虎卫 +3')
	assert.equal(formatTraitOutcomeDetail('revivedUnits', detail.traits[2].detail.revivedUnits), '复活兵力: 贪狼营 +35')
})

test('轮回副本全正式特性矩阵只展示后端实际触发项', () => {
	const bothSides = 'weiwu_tongyu yibing_touxi guicai_yice renzhu_shouhu shuiyan_qijun zhenhe_quanjun qimen_dunjia wolong_mouzhi xiliang_tuji laodang_yizhuang huoshao_lianying lianying_zengshang kurouji kurou_fanji'.split(' ')
	const attackerOnly = 'meiren meihuo_raozhen huchi_chongzhen sizhandaodi weizhen_zhenhe weizhen_xiaoyao wusheng_pojun wanren_nuhou baibu_chuanyang qibing_raohou xiaobawang_tieqi huogong meizhoulang_junlue'.split(' ')
	const defenderOnly = 'mouding_houfa huzhu_xuezhan dunzhen_fangyu longdan_jiuyuan gushou_hanzhong jiangdong_gushou'.split(' ')
	const nonBattle = 'weiwu_haoling jixing_benxi huhu_shengwei shengui_zhicai rende wangzuo_zhicai neizheng_jingying qijin_qichu tianshen_xiafan jiangdong_haoling xiaobawang_zhuiji baiyi_dujiang baiyi_jixing kuairu_shandian xinyi_yonglie jinfan_jielue jinfan_qixi'.split(' ')
	const allTraitIds = [...bothSides, ...attackerOnly, ...defenderOnly, ...nonBattle]
	assert.equal(new Set(allTraitIds).size, 50)

	const attackTraits = [...bothSides, ...attackerOnly].map((traitId) => ({ traitId, ownerSide: 'primary', ownerRole: 'attacker', generalId: 'g1' }))
	const attackDetail = { sourceType: 'dungeon', viewType: 'attack', battleType: 'dungeon_reincarnation_attack', traits: attackTraits }
	assert.equal(attackTraits.length, 27)
	assert.deepEqual(attackTraits.map((trait) => resolveReportTraitDisplaySide(attackDetail, trait)), Array(27).fill('primary'))
	assert.equal(attackTraits.every((trait) => getTraitMeta(trait.traitId).name !== trait.traitId && getTraitMeta(trait.traitId).trigger.length > 0), true)

	const defenseTraits = [...bothSides, ...defenderOnly].map((traitId) => ({ traitId, ownerSide: 'secondary', ownerRole: 'defender', generalId: 'g2' }))
	const defenseDetail = { sourceType: 'dungeon', viewType: 'defense', battleType: 'dungeon_reincarnation_defense', traits: defenseTraits }
	assert.equal(defenseTraits.length, 20)
	assert.deepEqual(defenseTraits.map((trait) => resolveReportTraitDisplaySide(defenseDetail, trait)), Array(20).fill('secondary'))

	const snapshotOnly = {
		sourceType: 'dungeon', viewType: 'attack', traits: [],
		primarySide: { generals: [{ id: 'g1', traits: allTraitIds.map((traitId) => ({ traitId })) }] },
	}
	assert.equal(snapshotOnly.primarySide.generals[0].traits.length, 50)
	assert.equal(hasTraitEntries(snapshotOnly), false)
})

test('轮回攻防无将领时不从城内主将快照补造特性', () => {
	for (const viewType of ['attack', 'defense']) {
		const detail = {
			sourceType: 'dungeon', viewType, battleType: `dungeon_reincarnation_${viewType}`,
			traits: [],
			primarySide: { role: 'attacker', generals: [] },
			secondarySide: { role: 'defender', generals: [] },
			rewards: { generalExp: 0, generalLevelBefore: 0, generalLevelAfter: 0 },
		}
		assert.equal(hasTraitEntries(detail), false)
		assert.equal(detail.primarySide.generals.length, 0)
		assert.equal(detail.secondarySide.generals.length, 0)
		assert.equal(detail.rewards.generalExp, 0)
	}
})

test('郭嘉在轮回攻防平局中展示真实阵亡和复活结果', () => {
	for (const [generalId, generalName, traitId, traitName] of [
		['guojia', '郭嘉', 'guicai_yice', '鬼才遗策'],
	] as const) {
		for (const viewType of ['attack', 'defense'] as const) {
			const playerSide = {
				role: viewType === 'attack' ? 'attacker' : 'defender', faction: 'wei', power: 1000,
				generals: [{ id: generalId, name: generalName, level: 1, traits: [{ traitId, name: traitName }] }],
				units: [{ unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 100, dispatched: 100, lost: 100, survived: 22 }],
			}
			const enemySide = {
				role: viewType === 'attack' ? 'defender' : 'attacker', faction: 'shu', power: 1000,
				generals: [],
				units: [{ unitType: 'shuInfantry', unitName: '蜀步兵', amountBefore: 100, dispatched: 100, lost: 100, survived: 0 }],
			}
			const detail = normalizeBattleReportDetail({
				id: `reincarnation-draw-${traitId}-${viewType}`,
				playerId: 'player', ownerPlayerId: 'player', viewType, sourceType: 'dungeon',
				battleType: `dungeon_reincarnation_${viewType}`, type: viewType, result: 'draw',
				defenderRevealed: true, rewards: {}, read: true, createdAt: '2026-07-20T18:00:00Z',
				detail: {
					id: `reincarnation-draw-${traitId}-${viewType}`,
					sourceType: 'dungeon', viewType, battleType: `dungeon_reincarnation_${viewType}`,
					result: 'draw', winnerSide: 'draw', ownerSide: viewType === 'attack' ? 'attacker' : 'defender',
					primarySide: viewType === 'attack' ? playerSide : enemySide,
					secondarySide: viewType === 'attack' ? enemySide : playerSide,
					rewards: { generalExp: 100, resources: {} },
					traits: [{
						traitId, traitName, generalId,
						ownerSide: viewType === 'attack' ? 'primary' : 'secondary',
						ownerRole: viewType === 'attack' ? 'attacker' : 'defender',
						detail: { effectRate: 0.22, actualLostUnits: { weiInfantry: 100 }, revivedUnits: { weiInfantry: 22 }, totalRevived: 22 },
					}],
				},
			})
			const ownedSide = viewType === 'attack' ? detail.primarySide : detail.secondarySide
			assert.equal(ownedSide?.generals?.[0].traits?.[0].traitId, traitId)
			assert.deepEqual(ownedSide?.units[0], {
				unitType: 'weiInfantry', unitName: '魏步兵', amountBefore: 100, dispatched: 100, lost: 100, survived: 22,
			})
			assert.equal(detail.primarySide.power, 1000)
			assert.equal(detail.secondarySide?.power, 1000)
			assert.equal(detail.rewards?.generalExp, 100)
			assert.equal(hasTraitEntries(detail), true)
			assert.equal(detail.traits[0].detail.revivedUnits.weiInfantry, 22)
		}
	}
})

test('敌方将领情报隐藏后只展示我方真实特性且不补回旧快照', () => {
	const detail = {
		sourceType: 'player_city',
		viewType: 'attack',
		visibility: {
			showEnemyRemainingUnits: true,
			showEnemyResources: false,
			showEnemyGenerals: false,
			showEnemyCityDefense: false,
		},
		primarySide: { role: 'attacker', playerId: 'attacker', generals: [{ id: 'caocao' }] },
		secondarySide: { role: 'defender', playerId: 'defender', generals: [] },
		traits: [{
			traitId: 'weiwu_tongyu',
			ownerSide: 'primary',
			ownerRole: 'attacker',
			generalId: 'caocao',
		}],
	}

	assert.equal(detail.secondarySide.generals.length, 0)
	assert.deepEqual(detail.traits.map((trait) => trait.traitId), ['weiwu_tongyu'])
	assert.equal(reportTraitBelongsToSide(detail, detail.traits[0], detail.primarySide, 'primary'), true)
	assert.equal(reportTraitBelongsToSide(detail, detail.traits[0], detail.secondarySide, 'secondary'), false)
	assert.equal(hasTraitEntries(detail), true)
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
  assert.equal(isReportShareToken('br_3f71a0e00843324f1977875f902085673c57c113f2f3eb83'), true)
  assert.equal(isReportShareToken('br_pvp_atk_internal'), false)
  const sharePageSource = readFileSync(new URL('../src/pages/report/ReportSharePage.tsx', import.meta.url), 'utf8')
  assert.match(sharePageSource, /isReportShareToken\(reportId\)/)
  assert.doesNotMatch(sharePageSource, /getReport\(reportId, activePlayerId\)\.catch/)
})

test('旧黄巾防守特性从错误主侧兼容映射到防守方', () => {
  const units = { faction: 'wei', units: { wei: { weiInfantry: { name: '魏步兵' } } } }
  assert.equal(resolveReportTraitDisplaySide(
    { sourceType: 'yellow_turban', viewType: 'defense' },
    { ownerSide: 'primary' },
  ), 'secondary')
  assert.equal(resolveReportTraitDisplaySide(
    { sourceType: 'player_city', viewType: 'attack' },
    { ownerSide: 'primary' },
  ), 'primary')
  assert.equal(resolveReportTraitDisplaySide(
    { sourceType: 'yellow_turban', viewType: 'defense' },
    { ownerSide: 'secondary', ownerRole: 'defender', generalId: 'simayi' },
  ), 'secondary')
  assert.equal(formatTraitOutcomeDetail('attackReductionRate', 0.1), '设计攻击降低: 10%')
  assert.equal(formatTraitOutcomeDetail('attackModifiedUnits', { weiInfantry: -1 }, units), '实际攻击修正: 魏步兵 -1')
  assert.equal(formatTraitOutcomeDetail('defenseBonusRate', 0.5), '设计防御加成: 50%')
  assert.equal(formatTraitOutcomeDetail('infantryDefenseModifiedUnits', { weiInfantry: 5 }, units), '实际步防修正: 魏步兵 +5')
  assert.equal(formatTraitOutcomeDetail('cavalryDefenseModifiedUnits', { weiInfantry: 4 }, units), '实际骑防修正: 魏步兵 +4')
})

test('孙权黄巾守城拥有双特性但真实时间线只展示江东固守', () => {
	const detail = {
		sourceType: 'yellow_turban', viewType: 'defense',
		traits: [{ traitId: 'jiangdong_gushou', traitName: '江东固守', ownerSide: 'secondary', ownerRole: 'defender', generalId: 'sunquan' }],
		secondarySide: {
			role: 'defender',
			generals: [{ id: 'sunquan', name: '孙权', traits: [{ traitId: 'jiangdong_haoling' }, { traitId: 'jiangdong_gushou' }] }],
		},
	}
	assert.deepEqual(detail.traits.map((trait) => resolveReportTraitDisplaySide(detail, trait)), ['secondary'])
	assert.deepEqual(detail.traits.map((trait) => trait.traitId), ['jiangdong_gushou'])
	assert.equal(detail.secondarySide.generals[0].traits.some((trait) => trait.traitId === 'jiangdong_haoling'), true)
	assert.equal(getTraitMeta('jiangdong_gushou').trigger, '防守/增援战斗前')
	assert.equal(getTraitMeta('jiangdong_haoling').trigger, '掠夺结算时')
})

test('军情详情等待完整接口响应后再渲染', () => {
  const source = readFileSync(new URL('../src/pages/news/NewsPage.tsx', import.meta.url), 'utf8')
  assert.doesNotMatch(source, /setSelectedReport\(report\)\s*\n\s*if \(!report\.read/)
  assert.match(source, /setLoadingReportId\(report\.id\)/)
  assert.match(source, /加载战报详情中/)
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
  const detailSource = readFileSync(new URL('../src/pages/news/components/BattleReportDetail.tsx', import.meta.url), 'utf8')
  assert.match(detailSource, /unitName: current\.unitName \|\| unitsConfig/)
})

test('将领携带特性快照不能代替本场真实触发时间线', () => {
	const randomTraitIds = [
		'yibing_touxi', 'huchi_chongzhen', 'sizhandaodi', 'weizhen_zhenhe',
		'shuiyan_qijun', 'zhenhe_quanjun', 'longdan_jiuyuan', 'xiliang_tuji',
		'baibu_chuanyang', 'qibing_raohou', 'jiangdong_gushou', 'xiaobawang_zhuiji',
		'huoshao_lianying', 'baiyi_dujiang', 'kuairu_shandian', 'kurouji',
	]
	const detailWithGeneralSnapshot = {
		primarySide: {
			generals: [{
				id: 'random_general', name: '随机特性测试将领',
				traits: randomTraitIds.map((traitId) => ({ traitId, name: getTraitMeta(traitId).name })),
			}],
		},
		traits: [],
	}
	assert.equal(detailWithGeneralSnapshot.primarySide.generals[0].traits.length, 16)
	assert.equal(hasTraitEntries(detailWithGeneralSnapshot), false)
  const detailWithoutGeneral = { traits: [], primarySide: { generals: [] }, secondarySide: { generals: [] } }
  assert.equal(hasTraitEntries(detailWithoutGeneral), false)
  const detailWithReturnedDefender = {
    traits: [{ traitId: 'jiangdong_gushou', traitName: '江东固守', ownerSide: 'secondary', generalId: 'sunquan' }],
    secondarySide: { generals: [{ id: 'sunquan', name: '孙权' }] },
  }
  assert.equal(hasTraitEntries(detailWithReturnedDefender), true)
})

test('增援配置变更后只按派出快照归属和后端真实时间线展示', () => {
	const reinforcementSide = {
		role: 'reinforcement', playerId: 'helper_wei',
		generals: [{ id: 'caocao', name: '曹操', traits: [{
			traitId: 'weiwu_tongyu', name: '魏武统御', params: { defenseBonusRate: 0.1 },
			allowedSides: ['attacker', 'defender', 'reinforcement'], allowedScenes: ['attack'],
		}] }],
	}
	const actualTrait = {
		traitId: 'weiwu_tongyu', traitName: '魏武统御', ownerSide: 'reinforcement', ownerRole: 'reinforcement',
		ownerPlayerId: 'helper_wei', generalId: 'caocao', detail: { defenseBonusRate: 0.1, infantryDefenseModifiedUnits: { huWei: 1 } },
	}
	const dispatchedEnabled = { sourceType: 'player_city', viewType: 'defense', traits: [actualTrait] }
	assert.equal(hasTraitEntries(dispatchedEnabled), true)
	assert.equal(reportTraitBelongsToSide(dispatchedEnabled, actualTrait, reinforcementSide, 'reinforcement'), true)

	const dispatchedDisabled = { sourceType: 'player_city', viewType: 'defense', traits: [], secondarySide: reinforcementSide }
	assert.equal(dispatchedDisabled.secondarySide.generals[0].traits.length, 1)
	assert.equal(dispatchedDisabled.secondarySide.generals[0].traits[0].params.defenseBonusRate, 0.1)
	assert.deepEqual(dispatchedDisabled.secondarySide.generals[0].traits[0].allowedSides, ['attacker', 'defender', 'reinforcement'])
	assert.equal(hasTraitEntries(dispatchedDisabled), false)
})

test('黄巾协防增援沿用派出快照但仍只展示后端真实时间线', () => {
	const reinforcementSide = {
		role: 'reinforcement', playerId: 'yellow_turban_helper',
		generals: [{ id: 'caocao', traits: [{ traitId: 'weiwu_tongyu', params: { defenseBonusRate: 0.1 } }] }],
	}
	const actualTrait = {
		traitId: 'weiwu_tongyu', traitName: '魏武统御', ownerSide: 'reinforcement', ownerRole: 'reinforcement',
		ownerPlayerId: 'yellow_turban_helper', generalId: 'caocao', detail: { defenseBonusRate: 0.1, infantryDefenseModifiedUnits: { huWei: 1 } },
	}
	const triggeredDetail = { sourceType: 'yellow_turban', viewType: 'defense', traits: [actualTrait] }
	assert.equal(reportTraitBelongsToSide(triggeredDetail, actualTrait, reinforcementSide, 'reinforcement'), true)
	assert.equal(hasTraitEntries(triggeredDetail), true)

	const snapshotOnlyDetail = { sourceType: 'yellow_turban', viewType: 'defense', traits: [], secondarySide: reinforcementSide }
	assert.equal(snapshotOnlyDetail.secondarySide.generals[0].traits[0].params.defenseBonusRate, 0.1)
	assert.equal(hasTraitEntries(snapshotOnlyDetail), false)
})

test('PVP 行军途中攻方关闭守方开启后只展示结算时真实特性', () => {
	const detail = {
		sourceType: 'player_city', viewType: 'attack',
		primarySide: { role: 'attacker', playerId: 'attacker', generals: [{ id: 'caocao', traits: [] }] },
		secondarySide: { role: 'defender', playerId: 'defender', generals: [{ id: 'sunquan', traits: [{ traitId: 'jiangdong_gushou' }] }] },
		traits: [{
			traitId: 'jiangdong_gushou', traitName: '江东固守', ownerSide: 'secondary', ownerRole: 'defender',
			ownerPlayerId: 'defender', generalId: 'sunquan', detail: { defenseBonusRate: 0.5, infantryDefenseModifiedUnits: { shuInfantry: 5 } },
		}],
	}
	assert.equal(detail.primarySide.generals[0].traits.some((trait) => trait.traitId === 'huogong'), false)
	assert.equal(reportTraitBelongsToSide(detail, detail.traits[0], detail.primarySide, 'primary'), false)
	assert.equal(reportTraitBelongsToSide(detail, detail.traits[0], detail.secondarySide, 'secondary'), true)
	assert.deepEqual(detail.traits.map((trait) => trait.traitId), ['jiangdong_gushou'])
})

test('黄巾来袭途中主将配置变化后只展示到达时真实防御特性', () => {
	const enabledDetail = {
		sourceType: 'yellow_turban', viewType: 'defense',
		secondarySide: { role: 'defender', playerId: 'sunquan_owner', generals: [{ id: 'sunquan', traits: [{ traitId: 'jiangdong_gushou' }] }] },
		traits: [{
			traitId: 'jiangdong_gushou', traitName: '江东固守', ownerSide: 'secondary', ownerRole: 'defender',
			ownerPlayerId: 'sunquan_owner', generalId: 'sunquan', detail: { defenseBonusRate: 0.5, infantryDefenseModifiedUnits: { wuInfantry: 5 } },
		}],
	}
	assert.equal(reportTraitBelongsToSide(enabledDetail, enabledDetail.traits[0], enabledDetail.secondarySide, 'secondary'), true)
	assert.equal(hasTraitEntries(enabledDetail), true)

	const disabledDetail = {
		sourceType: 'yellow_turban', viewType: 'defense', traits: [],
		secondarySide: { role: 'defender', playerId: 'sunquan_owner', generals: [{ id: 'sunquan', traits: [] }] },
	}
	assert.equal(disabledDetail.secondarySide.generals[0].traits.length, 0)
	assert.equal(hasTraitEntries(disabledDetail), false)
})

test('黄巾来袭途中主将离城或归城只展示结算时真实快照', () => {
	const awayDetail = {
		sourceType: 'yellow_turban', viewType: 'defense', traits: [],
		secondarySide: { role: 'defender', playerId: 'sunquan_owner', generals: [] },
		rewards: { generalExp: 0, generalLevelBefore: 0, generalLevelAfter: 0 },
	}
	assert.equal(awayDetail.secondarySide.generals.length, 0)
	assert.equal(awayDetail.rewards.generalExp, 0)
	assert.equal(hasTraitEntries(awayDetail), false)

	const returnedTrait = {
		traitId: 'jiangdong_gushou', traitName: '江东固守', ownerSide: 'secondary', ownerRole: 'defender',
		ownerPlayerId: 'sunquan_owner', generalId: 'sunquan',
		detail: { defenseBonusRate: 0.5, infantryDefenseModifiedUnits: { shuInfantry: 5 }, cavalryDefenseModifiedUnits: { shuInfantry: 4 } },
	}
	const returnedDetail = {
		sourceType: 'yellow_turban', viewType: 'defense', traits: [returnedTrait],
		secondarySide: { role: 'defender', playerId: 'sunquan_owner', generals: [{ id: 'sunquan', name: '孙权' }] },
		rewards: { generalExp: 60, generalLevelBefore: 1, generalLevelAfter: 1 },
	}
	assert.equal(returnedDetail.secondarySide.generals[0].id, 'sunquan')
	assert.equal(returnedDetail.rewards.generalExp, 60)
	assert.equal(reportTraitBelongsToSide(returnedDetail, returnedTrait, returnedDetail.secondarySide, 'secondary'), true)
	assert.equal(hasTraitEntries(returnedDetail), true)
})

test('黄巾来袭途中换将后只展示新主将战前快照和战后升级', () => {
	const trait = {
		traitId: 'dunzhen_fangyu', traitName: '盾阵防御', ownerSide: 'secondary', ownerRole: 'defender',
		ownerPlayerId: 'general_change_owner', generalId: 'xiahouyuan',
		detail: { defenseBonusRate: 0.3, infantryDefenseModifiedUnits: { weiInfantry: 3 }, cavalryDefenseModifiedUnits: { weiInfantry: 2 }, triggerChance: 0.6 },
	}
	const detail = {
		sourceType: 'yellow_turban', viewType: 'defense', traits: [trait],
		secondarySide: {
			role: 'defender', playerId: 'general_change_owner', power: 1339,
			generals: [{ id: 'xiahouyuan', name: '夏侯渊', level: 1, traits: [{ traitId: 'jixing_benxi', params: { unitAttackFlat: 18, unitSpeedFlat: 5 } }, { traitId: 'dunzhen_fangyu' }] }],
		},
		rewards: { generalExp: 100, generalLevelBefore: 1, generalLevelAfter: 2 },
	}
	assert.equal(detail.secondarySide.generals.some((general) => general.id === 'simayi'), false)
	assert.equal(detail.secondarySide.generals[0].level, 1)
	assert.deepEqual(detail.rewards, { generalExp: 100, generalLevelBefore: 1, generalLevelAfter: 2 })
	assert.equal(reportTraitBelongsToSide(detail, trait, detail.secondarySide, 'secondary'), true)
	assert.deepEqual(detail.traits.map((item) => item.traitId), ['dunzhen_fangyu'])
})

test('NPC 战后升级仍展示战前将领快照和实际特性时间线', () => {
	const trait = {
		traitId: 'meizhoulang_junlue', traitName: '美周郎军略', ownerSide: 'primary', ownerRole: 'attacker', generalId: 'zhouyu',
		detail: { attackBonusRate: 0.05, attackModifiedUnits: { weiInfantry: 1 } },
	}
	const detail = {
		sourceType: 'npc_city', viewType: 'attack', traits: [trait],
		primarySide: {
			role: 'attacker', power: 1100,
			generals: [{ id: 'zhouyu', name: '周瑜', level: 1, traits: [{ traitId: 'meizhoulang_junlue' }, { traitId: 'huogong' }] }],
		},
		rewards: { generalExp: 100, generalLevelBefore: 1, generalLevelAfter: 2 },
	}
	assert.equal(detail.primarySide.generals[0].level, 1)
	assert.deepEqual(detail.rewards, { generalExp: 100, generalLevelBefore: 1, generalLevelAfter: 2 })
	assert.equal(reportTraitBelongsToSide(detail, trait, detail.primarySide, 'primary'), true)
	assert.deepEqual(detail.traits.map((item) => item.traitId), ['meizhoulang_junlue'])
	assert.equal(hasTraitEntries(detail), true)
})

test('NPC 双城扫荡使用初始将领快照并单独展示累计升级', () => {
	const detail = {
		sourceType: 'npc_city', battleType: 'sweep', viewType: 'attack',
		primarySide: { role: 'attacker', power: 1000, generals: [{ id: 'zhouyu', name: '周瑜', level: 1 }] },
		rewards: { generalExp: 2, generalLevelBefore: 1, generalLevelAfter: 3 },
		traits: [],
	}
	assert.equal(detail.primarySide.generals[0].level, 1)
	assert.equal(detail.primarySide.power, 1000)
	assert.deepEqual(detail.rewards, { generalExp: 2, generalLevelBefore: 1, generalLevelAfter: 3 })
	assert.equal(hasTraitEntries(detail), false)
})

test('战报标准化只读扩展快照且保留后端权威嵌套数值', () => {
	const response = {
		id: 'extra-readonly', playerId: 'attacker', viewType: 'attack', sourceType: 'npc_city', battleType: 'sweep',
		type: 'sweep', result: 'attacker_victory', defenderRevealed: true, rewards: {}, read: true,
		detail: {
			id: 'extra-readonly', viewType: 'attack', sourceType: 'npc_city', battleType: 'sweep', result: 'attacker_victory',
			title: '扫荡战报', occurredAt: '2026-07-19T12:00:00Z',
			primarySide: { role: 'attacker', units: [], generals: [] },
			rewards: {}, traits: [],
			visibility: { showEnemyRemainingUnits: true, showEnemyResources: true, showEnemyGenerals: true, showEnemyCityDefense: true },
			extra: {
				pvp: { wall: { level: 10 }, reinforcementLosses: { rein_1: { shuInfantry: 2 } } },
				sweep: { defenders: [{ targetId: 'npc_1', resources: { wood: 5 } }] },
			},
			read: true,
		},
	}
	const before = JSON.stringify(response)
	const normalized = normalizeBattleReportDetail(response)

	assert.equal(JSON.stringify(response), before)
	assert.deepEqual(normalized.extra?.pvp?.reinforcementLosses, { rein_1: { shuInfantry: 2 } })
	assert.deepEqual(normalized.extra?.sweep?.defenders, [{ targetId: 'npc_1', resources: { wood: 5 } }])
})

test('标准战报只展示出动和阵亡，不重复顶部胜负和协防贡献', () => {
  const matrixSource = readFileSync(new URL('../src/pages/news/components/report-detail/UnitLossMatrix.tsx', import.meta.url), 'utf8')
  const headerSource = readFileSync(new URL('../src/pages/news/components/report-detail/BattleReportHeader.tsx', import.meta.url), 'utf8')
  const reinforcementSource = readFileSync(new URL('../src/pages/news/components/report-detail/ReportReinforcementContext.tsx', import.meta.url), 'utf8')
  assert.doesNotMatch(matrixSource, /剩余/)
  assert.doesNotMatch(headerSource, /BattleOutcomeSeal/)
  assert.doesNotMatch(reinforcementSource, /我的贡献/)
})

test('战报详情先聚合增援批次，且增援区块不显示结算行', () => {
  const detailSource = readFileSync(new URL('../src/pages/news/components/BattleReportDetail.tsx', import.meta.url), 'utf8')
  const participantSource = readFileSync(new URL('../src/pages/news/components/report-detail/BattleParticipantBlock.tsx', import.meta.url), 'utf8')
  assert.match(detailSource, /aggregateReinforcementSnapshots/)
  assert.match(detailSource, /generalExp=\{side\.generalExpGained\}/)
  assert.match(detailSource, /generalLevelBefore=\{side\.generalLevelBefore\}/)
  assert.match(detailSource, /generalLevelAfter=\{side\.generalLevelAfter\}/)
  assert.match(detailSource, /settlement="none"/)
  assert.match(participantSource, /武将经验 \+\{generalExp\.toLocaleString\(\)\}/)
  assert.match(participantSource, /generalLevelBefore/)
  assert.match(participantSource, /generalLevelAfter/)
  assert.match(participantSource, /Lv\.\$\{generalLevelBefore\} → Lv\.\$\{generalLevelAfter\}/)
  assert.match(detailSource, /trait\.summary\?\.trim\(\) === name\.trim\(\)/)
  assert.doesNotMatch(detailSource, /detail\.viewType === 'reinforcement' \? \[\] : buildReinforcementSides/)
  assert.match(detailSource, /attackerSide\.role === 'reinforcement' \? '增援方' : '攻击方'/)
  assert.match(detailSource, /reportTraitBelongsToSide/)
  assert.match(detailSource, /rewards=\{detail\.ownerSide === 'defender' \? undefined : detail\.rewards\}/)
  assert.match(detailSource, /rewards=\{detail\.ownerSide === 'defender' \? detail\.rewards : undefined\}/)
})
