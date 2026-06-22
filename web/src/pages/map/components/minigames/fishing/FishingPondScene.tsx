import type { CSSProperties, FC } from 'react'
import fishingPondBg from '@/assets/minigames/fishing/fishing-pond-bg.webp'
import fishermanActionSprites from '@/assets/minigames/fishing/fisherman-only-sprites.png'
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
  const handlePointerDown = () => {
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
  const isSweetSpot = castPower >= selectedBait.sweetStart && castPower <= selectedBait.sweetEnd
  const castPreviewX = 42 + castPower * 0.4
  const castPreviewY = 70 - castPower * 0.28
  const castColor = isSweetSpot ? '#22c55e' : castPower > selectedBait.sweetEnd ? '#ef4444' : '#38bdf8'
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
      <div className="absolute inset-0 bg-gradient-to-b from-transparent via-transparent to-slate-950/12" />
      <div className={`absolute inset-x-0 top-0 h-[32%] bg-gradient-to-b ${sceneTone} opacity-15`} />
      <div className="absolute inset-x-0 bottom-0 h-[48%] bg-gradient-to-t from-slate-950/18 via-cyan-950/5 to-transparent" />
      <div className="absolute bottom-[14%] left-[34%] h-[40%] w-[58%] rounded-[50%] bg-[radial-gradient(ellipse_at_center,rgba(255,255,255,0.16),rgba(14,165,233,0.04)_42%,transparent_68%)] opacity-70 animate-water-shimmer" />

      {phase === 'casting' && (
        <>
          <div
            className="absolute h-14 w-14 -translate-x-1/2 -translate-y-1/2 rounded-full border-2 border-white/60 bg-white/10 shadow-[0_0_18px_rgba(255,255,255,0.25)]"
            style={{ left: `${castPreviewX}%`, top: `${castPreviewY}%` }}
          />
          <div
            className="absolute h-2 w-2 -translate-x-1/2 -translate-y-1/2 rounded-full shadow-[0_0_12px_rgba(255,255,255,0.75)]"
            style={{ left: `${castPreviewX}%`, top: `${castPreviewY}%`, backgroundColor: castColor }}
          />
          <div
            className="absolute left-[31%] top-[62%] h-px origin-left rotate-[-23deg] bg-white/55"
            style={{ width: `${Math.max(80, castPower * 3.2)}px` }}
          />
        </>
      )}

      <div
        className="absolute bottom-[55%] left-[14%] h-[34%] aspect-[340/440] pointer-events-none [image-rendering:pixelated] sm:bottom-[27%] sm:left-[16%] md:bottom-[30%] md:left-[19%]"
        style={spriteFrameStyle(fishermanActionSprites, 6, fishermanFrame)}
      />

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
          <div className="rounded border border-amber-900/50 bg-amber-50/88 px-2 py-1 text-[10px] font-bold text-amber-950 shadow-[0_2px_8px_rgba(15,23,42,0.18)] backdrop-blur-sm">
            {selectedBait.name} · {selectedBait.cityGoldCost} 城金
          </div>
          <div className="hidden rounded border border-sky-950/40 bg-sky-50/88 px-2 py-1 text-[10px] font-bold text-sky-950 shadow-[0_2px_8px_rgba(15,23,42,0.18)] backdrop-blur-sm sm:block">
            蓄力框 {selectedBait.sweetStart}-{selectedBait.sweetEnd}%
          </div>
        </div>

        {phase === 'idle' && (
          <div className="mb-4 self-center rounded border border-slate-900/35 bg-white/82 px-3 py-2 text-center shadow-[0_6px_18px_rgba(15,23,42,0.18)] backdrop-blur-sm">
            <p className="text-[11px] font-black text-slate-900">{combo > 0 ? `连击 ${combo}，鱼群聚过来了` : '按住画面蓄力，松手投杆'}</p>
          </div>
        )}

        {phase === 'casting' && (
          <div className="mb-4 w-full max-w-[250px] self-end rounded border border-slate-900/40 bg-white/86 p-2.5 shadow-[0_8px_22px_rgba(15,23,42,0.2)] backdrop-blur-sm">
            <div className="mb-1.5 flex items-center justify-between gap-2">
              <p className="text-[11px] font-black text-slate-900">松手投杆</p>
              <p className="text-sm font-black" style={{ color: castColor }}>{Math.round(castPower)}%</p>
            </div>
            <div className="relative">
              <div className="h-3 w-full overflow-hidden rounded-full border border-slate-900/45 bg-slate-200">
                <div
                  className="h-full transition-all duration-[25ms]"
                  style={{
                    width: `${castPower}%`,
                    background: castColor,
                  }}
                />
              </div>
              <div
                className="pointer-events-none absolute top-0 bottom-0 border-x-2 border-green-900/80 bg-green-400/25"
                style={{ left: `${selectedBait.sweetStart}%`, width: `${selectedBait.sweetEnd - selectedBait.sweetStart}%` }}
              />
            </div>
            <div className="mt-1 flex justify-between px-0.5 text-[9px] text-slate-500">
              <span>弱</span>
              <span className="font-bold text-green-700">最佳 {selectedBait.sweetStart}-{selectedBait.sweetEnd}%</span>
              <span>强</span>
            </div>
          </div>
        )}

        {phase === 'waiting' && (
          <div className="mb-4 self-center rounded border border-slate-900/35 bg-white/84 px-3 py-2 text-center shadow-[0_6px_18px_rgba(15,23,42,0.18)] backdrop-blur-sm">
            <p className="text-xs font-black text-slate-900">
              {tensionLevel === 0 ? '鱼线入水...' : tensionLevel === 1 ? '水面微动...' : tensionLevel === 2 ? '鱼影靠近...' : '盯紧水面'}
            </p>
          </div>
        )}

        {phase === 'biting' && (
          <div className="mb-4 self-center rounded border border-red-900/45 bg-amber-100/90 px-4 py-2 text-center shadow-[0_8px_22px_rgba(127,29,29,0.22)] animate-pulse backdrop-blur-sm">
            <p className="text-sm font-black text-red-700">水面起涟漪！点击收杆</p>
          </div>
        )}

        {phase === 'reeling' && (
          <div className="mb-4 self-center rounded border border-slate-900/35 bg-white/84 px-5 py-3 text-center shadow-[0_6px_18px_rgba(15,23,42,0.18)] backdrop-blur-sm">
            <p className="text-xs font-black text-slate-900">收杆中...</p>
            <div className="mt-2 flex justify-center gap-1.5">
              {[0, 1, 2, 3].map(i => (
                <div key={i} className="h-2 w-2 bg-amber-500 animate-bounce" style={{ animationDelay: `${i * 0.12}s` }} />
              ))}
            </div>
          </div>
        )}

        {phase === 'escaped' && (
          <div className="mb-4 self-center rounded border border-slate-900/35 bg-white/86 px-5 py-3 text-center shadow-[0_6px_18px_rgba(15,23,42,0.18)] backdrop-blur-sm">
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
          <div className="mb-4 self-center rounded border border-slate-900/35 bg-white/86 px-5 py-3 text-center shadow-[0_6px_18px_rgba(15,23,42,0.18)] backdrop-blur-sm">
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
