/* 本文件渲染战报兵种出动、阵亡和剩余矩阵。 */
import type { FC } from 'react'
import type { BattleReportUnit } from '@/types/game'

interface UnitLossMatrixProps {
  title: string
  units?: BattleReportUnit[]
}

// UnitLossMatrix 在桌面显示横向矩阵，移动端显示兵种行列表。
const UnitLossMatrix: FC<UnitLossMatrixProps> = ({ title, units = [] }) => {
  const rows = units
  return (
    <section className="border-t border-[var(--color-border)]">
      <div className="flex items-center justify-between border-b border-[var(--color-border)] px-3 py-2">
        <h3 className="text-xs font-bold text-amber-500">{title}</h3>
      </div>
      {rows.length === 0 ? (
        <div className="px-3 py-4 text-xs text-[var(--color-text-muted)]">无参战兵种</div>
      ) : (
        <>
          <div className="hidden overflow-x-auto p-3 sm:block">
            <table className="w-full text-center text-[11px]">
              <thead className="text-[var(--color-text-muted)]">
                <tr>
                  <th className="min-w-20 py-1 text-left font-semibold">项目</th>
                  {rows.map((unit) => <th key={unit.unitType} className="min-w-16 px-2 py-1 font-semibold">{unit.unitName || unit.unitType}</th>)}
                </tr>
              </thead>
              <tbody>
                {[
                  ['出动', 'dispatched'],
                  ['阵亡', 'lost'],
                  ['剩余', 'survived'],
                ].map(([label, key]) => (
                  <tr key={key} className="border-t border-[var(--color-border)]">
                    <td className="py-1.5 text-left font-semibold text-[var(--color-text-secondary)]">{label}</td>
                    {rows.map((unit) => (
                      <td key={unit.unitType} className={key === 'lost' && unit.lost > 0 ? 'px-2 py-1.5 font-bold text-red-500' : 'px-2 py-1.5 text-[var(--color-text-primary)]'}>
                        {Number(unit[key as keyof BattleReportUnit] || 0).toLocaleString()}
                      </td>
                    ))}
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          <div className="space-y-1.5 p-3 sm:hidden">
            {rows.map((unit) => (
              <div key={unit.unitType} className="rounded-md border border-[var(--color-border)] bg-[var(--color-surface-dim)] px-2 py-1.5">
                <div className="text-[11px] font-bold text-[var(--color-text-primary)]">{unit.unitName || unit.unitType}</div>
                <div className="mt-1 grid grid-cols-3 gap-1 text-[10px] text-[var(--color-text-secondary)]">
                  <span>出动 {unit.dispatched.toLocaleString()}</span>
                  <span className={unit.lost > 0 ? 'font-bold text-red-500' : ''}>阵亡 {unit.lost.toLocaleString()}</span>
                  <span>剩余 {unit.survived.toLocaleString()}</span>
                </div>
              </div>
            ))}
          </div>
        </>
      )}
    </section>
  )
}

export default UnitLossMatrix
