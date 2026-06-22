import type { CSSProperties, FC, PointerEvent } from 'react'
import fishingPondBg from '@/assets/minigames/fishing/fishing-pond-bg.webp'
import fishermanActionSprites from '@/assets/minigames/fishing/fisherman-action-sprites.png'
import waterEffectsSprites from '@/assets/minigames/fishing/water-effects.png'
import type { BaitType, BiteSpot, Bubble, FishCatch, FishShadow, GamePhase } from './types'

interface FishingPondSceneProps {
  phase: GamePhase
  selectedBait: BaitType
  castPower: number
  combo: number
  tensionLevel: number
  bubbles: Bubble[]
  fishShadow: FishShadow
  biteSpot: BiteSpot
  catchResult: FishCatch | null
  showResult: boolean
  onCastDown: () => void
  onCastUp: () => void
  onReel: () => void
  onReset: () => void
}

const spriteFrameStyle = (imageUrl: string, frameCount: number, frameIndex: number): CSSProperties => ({
  backgroundImage: `url(${imageUrl})`,
  backgroundSize: `${frameCount * 100}% 100%`,
  backgroundPosition: `${frameCount <= 1 ? 0 : (frameIndex / (frameCount - 1)) * 100}% 0`,
  backgroundRepeat: 'no-repeat',
})

export const FishingPondScene: FC<FishingPondSceneProps> = ({
  phase,
  selectedBait,
  castPower,
  combo,
  tensionLevel,
  bubbles,
  fishShadow,
  biteSpot,
  catchResult,
  showResult,
  onCastDown,
  onCastUp,
  onReel,
  onReset,
}) => {
  const handlePointerDown = (_event: PointerEvent<HTMLDivElement>) => {
    if (phase === 'idle') onCastDown()
    else if (phase === 'biting') onReel()
  }

  const handlePointerUp = () => {
    if (phase === 'casting') onCastUp()
  }

  const sceneTone = catchResult?.rarity === 'legendary'
    ? 'from-amber-200 via-sky-200 to-cyan-300'
    : catchResult?.rarity === 'epic'
      ? 'from-purple-200 via-sky-200 to-cyan-300'
      : 'from-sky-200 via-sky-300 to-cyan-300'
  const fishermanFrame = phase === 'idle'
    ? 0
    : phase === 'casting'
    ? 1
    : phase === 'reeling'
      ? 5
      : phase === 'biting'
        ? 4
        : phase === 'waiting'
          ? 3
          : 0

  return (
    <div
      className="relative min-h-[420px] overflow-hidden rounded-xl border-4 border-slate-800 bg-[#86c06a] select-none touch-none shadow-[0_14px_0_#273449,0_28px_70px_rgba(15,23,42,0.25)] md:min-h-[470px]"
      onPointerDown={handlePointerDown}
      onPointerUp={handlePointerUp}
      onPointerCancel={handlePointerUp}
    >
      <img
        src={fishingPondBg}
        alt=""
        className="absolute inset-0 h-full w-full object-cover [image-rendering:auto]"
        draggable={false}
      />
      <div className="absolute inset-0 bg-gradient-to-b from-transparent via-transparent to-slate-950/8" />
      <div className={`absolute inset-x-0 top-0 h-[32%] bg-gradient-to-b ${sceneTone} opacity-15`} />
      <div className="absolute left-8 top-8 h-7 w-7 rounded-sm bg-[#ffe28a] shadow-[8px_0_0_#ffe28a,0_8px_0_#ffe28a,8px_8px_0_#f5c453]" />

      {/* Distant mountains */}
      <div className="absolute left-[-4%] top-[18%] h-20 w-52 bg-[#6fa389] [clip-path:polygon(0_100%,22%_30%,40%_100%,58%_18%,84%_100%,100%_100%)] opacity-80" />
      <div className="absolute right-[-6%] top-[16%] h-24 w-64 bg-[#5f947c] [clip-path:polygon(0_100%,18%_45%,34%_100%,52%_20%,74%_100%,100%_44%,100%_100%)] opacity-80" />

      {/* Pavilion */}
      <div className="absolute right-[10%] top-[27%] hidden h-20 w-24 sm:block">
        <div className="absolute left-2 top-0 h-4 w-20 bg-[#7c2d12] shadow-[6px_4px_0_#431407,-6px_4px_0_#431407]" />
        <div className="absolute left-5 top-4 h-9 w-2 bg-[#78350f]" />
        <div className="absolute right-5 top-4 h-9 w-2 bg-[#78350f]" />
        <div className="absolute bottom-4 left-3 h-3 w-18 bg-[#92400e]" />
      </div>

      <div className="absolute bottom-0 left-0 right-0 h-[68%] bg-[#6cad52]/15" />
      <div className="absolute bottom-0 left-0 h-[22%] w-[48%] bg-[#7a5b34]/35 shadow-[inset_0_8px_0_rgba(155,122,71,0.35)]" />
      <div className="absolute bottom-[9%] left-[31%] h-[64%] w-[67%] rounded-[45%] border-4 border-[#12394e]/40 bg-[#2996b7]/25 shadow-[inset_0_-22px_0_rgba(18,107,137,0.3),inset_0_14px_0_rgba(255,255,255,0.16),0_10px_0_rgba(15,79,102,0.28)]" />
      <div className="absolute bottom-[15%] left-[37%] h-[46%] w-[56%] rounded-[45%] bg-[repeating-linear-gradient(0deg,rgba(255,255,255,0.10)_0_2px,transparent_2px_20px)] opacity-25 animate-water-shimmer" />
      <div className="absolute bottom-[12%] left-[24%] h-12 w-20 rounded-t-full bg-[#7b5d38] shadow-[12px_8px_0_#5c4329]" />

      {/* Lotus / reeds */}
      <div className="absolute bottom-[28%] right-[13%] h-3 w-8 rounded-full bg-[#2f7d45] shadow-[24px_-12px_0_#2f7d45,-22px_10px_0_#3f9b57]" />
      <div className="absolute bottom-[40%] right-[20%] h-3 w-3 bg-[#f0a6c1] shadow-[4px_0_0_#f0a6c1,2px_-4px_0_#f7c1d4]" />
      <div className="absolute bottom-[25%] left-[4%] h-16 w-2 bg-[#315c2a] shadow-[8px_-8px_0_#315c2a,16px_2px_0_#315c2a,24px_-12px_0_#315c2a]" />

      <div
        className={`absolute bottom-[13%] left-[1%] h-32 w-[66%] pointer-events-none transition-transform duration-300 [image-rendering:pixelated] sm:bottom-[13%] sm:left-[2%] sm:h-40 sm:w-[60%] md:h-48 md:w-[58%] ${phase === 'casting' ? '-translate-y-1' : phase === 'reeling' ? '-translate-y-2' : ''}`}
        style={spriteFrameStyle(fishermanActionSprites, 6, fishermanFrame)}
      >
      </div>

      {(phase === 'waiting' || phase === 'biting' || phase === 'reeling') && (
        <>
          {phase === 'biting' && (
            <div
              className="absolute h-12 w-12 -translate-x-1/2 -translate-y-1/2 pointer-events-none animate-fishing-splash [image-rendering:pixelated]"
              style={{
                left: `${biteSpot.x}%`,
                top: `${biteSpot.y}%`,
                ...spriteFrameStyle(waterEffectsSprites, 6, 4),
              }}
            />
          )}
        </>
      )}

      {bubbles.map(b => (
        <div
          key={b.id}
          className="absolute animate-float-up pointer-events-none [image-rendering:pixelated]"
          style={{
            left: `${Math.max(42, b.x)}%`,
            bottom: '18%',
            width: Math.max(18, b.size * 2),
            height: Math.max(18, b.size * 2),
            animationDelay: `${b.delay}s`,
            animationDuration: '2s',
            ...spriteFrameStyle(waterEffectsSprites, 6, 5),
          }}
        />
      ))}

      {fishShadow.visible && (
        <div
          className="absolute top-[58%] transition-all duration-500 ease-in-out"
          style={{ left: `${fishShadow.x}%`, transform: 'translateX(-50%)' }}
        >
          <div
            className="h-12 w-24 opacity-70 animate-fish-shadow-glide [image-rendering:pixelated]"
            style={spriteFrameStyle(waterEffectsSprites, 6, 1)}
          />
        </div>
      )}

      {phase === 'biting' && (
        <div
          className="absolute z-20 h-20 w-20 -translate-x-1/2 -translate-y-1/2 cursor-pointer"
          style={{ left: `${biteSpot.x + 1}%`, top: `${biteSpot.y + 3}%` }}
        >
          <div
            className="absolute inset-0 scale-125 animate-fishing-ripple-soft opacity-75 [image-rendering:pixelated]"
            style={spriteFrameStyle(waterEffectsSprites, 6, 3)}
          />
          <div
            className="absolute inset-0 animate-fishing-splash opacity-95 [image-rendering:pixelated]"
            style={spriteFrameStyle(waterEffectsSprites, 6, 4)}
          />
        </div>
      )}

      {(catchResult?.rarity === 'legendary' || catchResult?.rarity === 'epic') && phase === 'caught' && (
        <div className={`absolute inset-0 z-10 pointer-events-none ${catchResult.rarity === 'legendary' ? 'bg-amber-300/20' : 'bg-purple-400/15'} animate-pulse`} />
      )}

      <div className="relative z-10 flex min-h-[420px] flex-col justify-between p-3 md:min-h-[470px] md:p-4">
        <div className="flex items-start justify-between gap-2">
          <div className="rounded-md border-2 border-slate-800 bg-white/80 px-2 py-1 text-[10px] font-bold text-slate-800 shadow-[3px_3px_0_#334155]">
            {selectedBait.name} · {selectedBait.cityGoldCost} 城金
          </div>
          <div className="hidden rounded-md border-2 border-slate-800 bg-white/80 px-2 py-1 text-[10px] font-bold text-slate-800 shadow-[3px_3px_0_#334155] sm:block">
            蓄力框 {selectedBait.sweetStart}-{selectedBait.sweetEnd}%
          </div>
        </div>

        {phase === 'idle' && (
          <div className="mb-5 self-center rounded-md border-4 border-slate-800 bg-white/90 px-5 py-3 text-center shadow-[5px_5px_0_#334155]">
            <p className="text-xs font-black text-slate-900">{combo > 0 ? `连击 ${combo}，鱼群聚过来了` : '按住钓场蓄力，松手投杆'}</p>
            <p className="mt-1 text-[10px] text-slate-500 md:hidden">手机端已简化显示，复杂版后续单独设计</p>
          </div>
        )}

        {phase === 'casting' && (
          <div className="mb-4 w-full max-w-[360px] self-center rounded-md border-4 border-slate-800 bg-white/90 p-3 shadow-[5px_5px_0_#334155]">
            <p className="mb-2 text-center text-xs font-black text-slate-900">蓄力中，松手投杆</p>
            <div className="relative">
              <div className="h-5 w-full overflow-hidden border-2 border-slate-900 bg-slate-200">
                <div
                  className="h-full transition-all duration-[25ms]"
                  style={{
                    width: `${castPower}%`,
                    background: castPower >= selectedBait.sweetStart && castPower <= selectedBait.sweetEnd ? '#22c55e' : castPower > selectedBait.sweetEnd ? '#ef4444' : '#38bdf8',
                  }}
                />
              </div>
              <div
                className="pointer-events-none absolute top-0 bottom-0 border-x-4 border-green-900 bg-green-400/25"
                style={{ left: `${selectedBait.sweetStart}%`, width: `${selectedBait.sweetEnd - selectedBait.sweetStart}%` }}
              />
            </div>
            <div className="mt-1 flex justify-between px-0.5 text-[9px] text-slate-500">
              <span>弱</span>
              <span className="font-bold text-green-700">最佳 {selectedBait.sweetStart}-{selectedBait.sweetEnd}%</span>
              <span>强</span>
            </div>
            <p className="mt-1 text-center text-lg font-black text-slate-900">{castPower}%</p>
          </div>
        )}

        {phase === 'waiting' && (
          <div className="mb-4 self-center rounded-md border-4 border-slate-800 bg-white/90 px-4 py-2 text-center shadow-[5px_5px_0_#334155]">
            <p className="text-xs font-black text-slate-900">
              {tensionLevel === 0 ? '鱼线入水...' : tensionLevel === 1 ? '水面微动...' : tensionLevel === 2 ? '鱼影靠近...' : '盯紧水面'}
            </p>
            <p className="mt-1 text-[10px] text-slate-500">涟漪出现后要立刻点击</p>
          </div>
        )}

        {phase === 'biting' && (
          <div className="mb-4 self-center rounded-md border-4 border-red-800 bg-amber-100 px-5 py-2 text-center shadow-[5px_5px_0_#7f1d1d] animate-pulse">
            <p className="text-sm font-black text-red-700">水面起涟漪！点击收杆</p>
            <p className="mt-1 text-[10px] text-red-500">超时涟漪会消散</p>
          </div>
        )}

        {phase === 'reeling' && (
          <div className="mb-4 self-center rounded-md border-4 border-slate-800 bg-white/90 px-5 py-3 text-center shadow-[5px_5px_0_#334155]">
            <p className="text-xs font-black text-slate-900">收杆中...</p>
            <div className="mt-2 flex justify-center gap-1.5">
              {[0, 1, 2, 3].map(i => (
                <div key={i} className="h-2 w-2 bg-amber-500 animate-bounce" style={{ animationDelay: `${i * 0.12}s` }} />
              ))}
            </div>
          </div>
        )}

        {phase === 'escaped' && (
          <div className="mb-4 self-center rounded-md border-4 border-slate-800 bg-white/90 px-5 py-3 text-center shadow-[5px_5px_0_#334155]">
            <p className="text-xs font-black text-slate-900">涟漪散了，鱼跑掉了</p>
            {combo > 0 && <p className="mt-1 text-[9px] text-red-600">连击中断</p>}
            <button
              type="button"
              onClick={onReset}
              className="mt-3 rounded-md border-2 border-slate-900 bg-blue-500 px-6 py-2 text-xs font-black text-white transition-all hover:bg-blue-600 cursor-pointer"
            >
              再来
            </button>
          </div>
        )}

        {phase === 'caught' && !showResult && catchResult && (
          <div className="mb-4 self-center rounded-md border-4 border-slate-800 bg-white/90 px-5 py-3 text-center shadow-[5px_5px_0_#334155]">
            <div className={`
              mx-auto flex h-16 w-16 items-center justify-center border-4 transition-all duration-500
              ${catchResult.rarity === 'legendary'
                ? 'border-amber-500 bg-amber-500/20 shadow-[0_0_40px_rgba(245,158,11,0.5)] animate-pulse'
                : catchResult.rarity === 'epic'
                  ? 'border-purple-500 bg-purple-500/20 shadow-[0_0_25px_rgba(168,85,247,0.4)] animate-pulse'
                  : catchResult.rarity === 'rare'
                    ? 'border-blue-500 bg-blue-500/20 shadow-[0_0_15px_rgba(59,130,246,0.3)]'
                    : 'border-slate-500 bg-slate-500/10'}
            `}>
              <span className={`text-3xl ${catchResult.rarity === 'legendary' ? 'animate-bounce' : ''}`}>
                {catchResult.emoji}
              </span>
            </div>
            <p className={`mt-2 text-sm font-black animate-pulse
              ${catchResult.rarity === 'legendary' ? 'text-amber-400' :
                catchResult.rarity === 'epic' ? 'text-purple-400' :
                  catchResult.rarity === 'rare' ? 'text-blue-400' :
                    'text-slate-700'}
            `}>
              {catchResult.rarity === 'legendary' ? '传说之物浮出水面...' :
                catchResult.rarity === 'epic' ? '史诗灵兽！' :
                  catchResult.rarity === 'rare' ? '稀有鱼种！' :
                    '钓到了'}
            </p>
          </div>
        )}
      </div>
    </div>
  )
}
