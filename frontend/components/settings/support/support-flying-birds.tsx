"use client"

import * as React from "react"

interface P5Image {
  drawingContext: CanvasRenderingContext2D
}

// Keep the local p5 surface limited to the APIs these scenes use.
interface P5Instance {
  HALF_PI: number
  PI: number
  ROUND: number
  TWO_PI: number
  canvas: HTMLCanvasElement
  height: number
  width: number
  atan2(x: number, y: number): number
  clear(): void
  constrain(value: number, minimum: number, maximum: number): number
  cos(angle: number): number
  createCanvas(width: number, height: number): void
  createImage(width: number, height: number): P5Image
  ellipse(x: number, y: number, width: number, height: number): void
  fill(...components: number[]): void
  image(image: P5Image, x: number, y: number): void
  lerp(start: number, stop: number, amount: number): number
  line(x1: number, y1: number, x2: number, y2: number): void
  loop(): void
  map(value: number, start1: number, stop1: number, start2: number, stop2: number): number
  millis(): number
  noFill(): void
  noLoop(): void
  noStroke(): void
  noTint(): void
  pixelDensity(density: number): void
  pop(): void
  pow(base: number, exponent: number): number
  push(): void
  radians(degrees: number): number
  random(minimum?: number, maximum?: number): number
  randomSeed(seed: number): void
  remove(): void
  resizeCanvas(width: number, height: number): void
  rotate(angle: number): void
  scale(x: number, y?: number): void
  setup?: () => void | Promise<void>
  sin(angle: number): number
  stroke(...components: number[]): void
  strokeCap(style: number): void
  strokeJoin(style: number): void
  strokeWeight(weight: number): void
  tint(...components: number[]): void
  translate(x: number, y: number): void
  windowResized?: () => void
  draw?: () => void
}

interface P5RuntimeInstance {
  loop(): void
  noLoop(): void
  remove(): void
}

interface P5Constructor {
  new (sketch: (p: P5Instance) => void, node?: HTMLElement): P5RuntimeInstance
}

let p5RuntimeLoader: Promise<P5Constructor> | null = null

function loadP5Runtime() {
  if (p5RuntimeLoader) return p5RuntimeLoader

  // The managed dependency keeps the lazy canvas runtime versioned with the app.
  p5RuntimeLoader = import("p5/lib/p5.js")
    .then(({ default: P5 }) => P5 as unknown as P5Constructor)
    .catch((error: unknown) => {
      p5RuntimeLoader = null
      throw error
    })

  return p5RuntimeLoader
}

const BIRD_ASSETS = {
  body: "/images/support/birds/body.svg",
  wingL: "/images/support/birds/wing-l.svg",
  wingR: "/images/support/birds/wing-r.svg",
}

function canUseP5Canvas() {
  return (
    typeof window !== "undefined" &&
    typeof document !== "undefined" &&
    !/jsdom/i.test(window.navigator.userAgent) &&
    typeof document.createElement("canvas").getContext === "function"
  )
}

function createSwallowSketch(host: HTMLDivElement) {
  return (p: P5Instance) => {
    const flockSize = 12
    const flightDurationMin = 4800
    const flightDurationMax = 6200
    const staggerSpan = 1000
    const birdScale = 0.4
    const wingFlapSpeed = 0.22
    const loopInterval = 5500
    const tailParticleCount = 12
    const tailTotalLength = 90
    const tailDamping = 0.94
    const tailConstraintIterations = 5

    let images = { body: null as P5Image | null, wingL: null as P5Image | null, wingR: null as P5Image | null }
    let swallows: SwallowBird[] = []
    let flockPhase: "idle" | "flying" | "done" = "idle"
    let flockTimer = 0
    let imagesReady = false
    const readBirdColor = (): [number, number, number] => {
      const root = document.documentElement
      return root.classList.contains("dark") || root.dataset.theme === "dark" ? [255, 255, 255] : [45, 45, 68]
    }
    let birdColor = readBirdColor()

    class SwallowBird {
      startX = p.width + p.random(30, 300)
      startY = p.random(60, Math.max(60, p.height - 60))
      exitX = p.random(-400, -50)
      exitY = p.random(60, Math.max(60, p.height - 60))
      cpX = 0
      cpY = 0
      startDelay: number
      flightDuration = p.random(flightDurationMin, flightDurationMax)
      birthTime = 0
      x = this.startX
      y = this.startY
      prevX = this.x
      prevY = this.y
      opacity = 0
      done = false
      wingPhase = p.random(p.TWO_PI)
      wingTime = p.random(p.TWO_PI)
      scale = birdScale * p.random(0.8, 1.2)
      tailL: Array<{ x: number; y: number; px: number; py: number }> = []
      tailR: Array<{ x: number; y: number; px: number; py: number }> = []

      constructor(index: number) {
        const middleX = (this.startX + this.exitX) / 2
        const middleY = (this.startY + this.exitY) / 2
        const dx = this.exitX - this.startX
        const dy = this.exitY - this.startY
        const distance = Math.hypot(dx, dy) || 1
        const bow = distance * p.random(0.2, 0.4)
        this.cpX = middleX + (dy / distance) * bow + p.random(-50, 50)
        this.cpY = middleY - (dx / distance) * bow + p.random(-50, 50)
        this.startDelay = (index / Math.max(flockSize - 1, 1)) * staggerSpan
        this.initializeTail()
      }

      initializeTail() {
        const segmentLength = tailTotalLength / (tailParticleCount - 1)
        for (const fork of [this.tailL, this.tailR]) {
          const angle = p.HALF_PI + (fork === this.tailL ? p.radians(8) : -p.radians(8))
          for (let index = 0; index < tailParticleCount; index += 1) {
            const distance = index * segmentLength
            fork.push({
              x: 67 + p.cos(angle) * distance,
              y: 54 + p.sin(angle) * distance,
              px: 67 + p.cos(angle) * distance,
              py: 54 + p.sin(angle) * distance,
            })
          }
        }
      }

      updateTail() {
        const segmentLength = tailTotalLength / (tailParticleCount - 1)
        for (const fork of [this.tailL, this.tailR]) {
          const rootX = fork[0].x
          const rootY = fork[0].y
          fork[0].px = fork[0].x
          fork[0].py = fork[0].y
          fork[0].x = 67
          fork[0].y = 54
          const rootDx = fork[0].x - rootX
          const rootDy = fork[0].y - rootY

          for (let index = 1; index < fork.length; index += 1) {
            const point = fork[index]
            const velocityX = (point.x - point.px) * tailDamping
            const velocityY = (point.y - point.py) * tailDamping
            point.px = point.x
            point.py = point.y
            point.x += velocityX + rootDx * 0.3
            point.y += velocityY + rootDy * 0.3
          }

          for (let iteration = 0; iteration < tailConstraintIterations; iteration += 1) {
            for (let index = 0; index < fork.length - 1; index += 1) {
              const start = fork[index]
              const end = fork[index + 1]
              const dx = end.x - start.x
              const dy = end.y - start.y
              const distance = Math.hypot(dx, dy) || 0.001
              const correction = ((distance - segmentLength) / distance) * 0.5
              if (index > 0) {
                start.x += dx * correction
                start.y += dy * correction
              }
              if (index < fork.length - 2) {
                end.x -= dx * correction
                end.y -= dy * correction
              }
            }
          }
        }
      }

      update() {
        const elapsed = p.millis() - this.birthTime - this.startDelay
        if (elapsed < 0) {
          this.opacity = 0
          this.updateTail()
          return
        }

        this.opacity = p.constrain(p.map(elapsed, 0, 200, 0, 255), 0, 255)
        const rawProgress = p.constrain(elapsed / this.flightDuration, 0, 1)
        this.prevX = this.x
        this.prevY = this.y
        if (rawProgress >= 1) {
          this.done = true
          this.x = this.exitX
          this.y = this.exitY
        } else {
          const progress = 1 - p.pow(1 - rawProgress, 2.8)
          const remaining = 1 - progress
          this.x = remaining * remaining * this.startX + 2 * remaining * progress * this.cpX + progress * progress * this.exitX
          this.y = remaining * remaining * this.startY + 2 * remaining * progress * this.cpY + progress * progress * this.exitY
        }
        this.wingTime += wingFlapSpeed
        this.updateTail()
      }

      display() {
        if (this.opacity <= 0 || this.done || !images.body || !images.wingL || !images.wingR) return
        p.push()
        p.translate(this.x, this.y)
        const dx = this.x - this.prevX
        const dy = this.y - this.prevY
        p.rotate(Math.abs(dx) < 0.01 && Math.abs(dy) < 0.01 ? -p.PI / 4 : p.atan2(dx, -dy))
        p.translate(-75, -28)
        p.scale(this.scale)
        const wingScale = p.lerp(0.2, 1, (p.sin(this.wingTime + this.wingPhase) + 1) / 2)

        p.push()
        p.translate(68, 0)
        p.scale(wingScale, 1)
        p.tint(...birdColor, this.opacity)
        p.image(images.wingR, 0, 0)
        p.pop()
        p.push()
        p.translate(66, 0)
        p.scale(wingScale, 1)
        p.tint(...birdColor, this.opacity)
        p.image(images.wingL, -66, 0)
        p.pop()
        p.push()
        p.translate(58, -14.5)
        p.tint(...birdColor, this.opacity)
        p.image(images.body, 0, 0)
        p.pop()
        p.noTint()
        p.noStroke()
        for (const fork of [this.tailL, this.tailR]) {
          for (let index = 0; index < fork.length - 1; index += 1) {
            const start = fork[index]
            const end = fork[index + 1]
            const alpha = this.opacity * (1 - index / (fork.length - 1))
            p.stroke(...birdColor, alpha)
            p.strokeWeight(1.1)
            p.line(start.x, start.y, end.x, end.y)
          }
        }
        p.pop()
      }
    }

    const loadSvg = async (url: string, width: number, height: number) => {
      const response = await fetch(url)
      if (!response.ok) throw new Error(`Failed to fetch ${url}`)
      const blobUrl = URL.createObjectURL(new Blob([await response.text()], { type: "image/svg+xml" }))
      const image = await new Promise<HTMLImageElement>((resolve, reject) => {
        const nativeImage = new Image()
        nativeImage.onload = () => resolve(nativeImage)
        nativeImage.onerror = reject
        nativeImage.src = blobUrl
      })
      URL.revokeObjectURL(blobUrl)
      const p5Image = p.createImage(width, height)
      const drawingContext = p5Image.drawingContext
      drawingContext.drawImage(image, 0, 0, width, height)
      drawingContext.globalCompositeOperation = "source-in"
      drawingContext.fillStyle = "#ffffff"
      drawingContext.fillRect(0, 0, width, height)
      drawingContext.globalCompositeOperation = "source-over"
      return p5Image
    }

    const startFlock = () => {
      swallows = Array.from({ length: flockSize }, (_, index) => new SwallowBird(index))
      flockTimer = p.millis()
      for (const bird of swallows) bird.birthTime = flockTimer
      flockPhase = "flying"
    }

    p.setup = async () => {
      const bounds = host.getBoundingClientRect()
      p.pixelDensity(Math.min(window.devicePixelRatio || 1, 2))
      p.createCanvas(Math.max(1, bounds.width), Math.max(1, bounds.height))
      p.canvas.setAttribute("aria-hidden", "true")
      p.canvas.className = "size-full"
      try {
        const [body, wingL, wingR] = await Promise.all([
          loadSvg(BIRD_ASSETS.body, 17, 69),
          loadSvg(BIRD_ASSETS.wingL, 66, 28),
          loadSvg(BIRD_ASSETS.wingR, 66, 28),
        ])
        images = { body, wingL, wingR }
        imagesReady = true
      } catch {
        p.noLoop()
      }
    }

    p.draw = () => {
      p.clear()
      if (!imagesReady || document.hidden) return
      birdColor = readBirdColor()
      if (flockPhase === "idle" && p.millis() > 600) startFlock()
      if (flockPhase === "flying") {
        for (const bird of swallows) {
          bird.update()
          bird.display()
        }
        if (swallows.every((bird) => bird.done)) {
          flockPhase = "done"
          flockTimer = p.millis()
        }
      }
      if (flockPhase === "done" && p.millis() - flockTimer > loopInterval) flockPhase = "idle"
    }

    p.windowResized = () => {
      const bounds = host.getBoundingClientRect()
      p.resizeCanvas(Math.max(1, bounds.width), Math.max(1, bounds.height))
    }
  }
}

export function SupportFlyingBirds({ reducedMotion = false }: { reducedMotion?: boolean }) {
  const hostRef = React.useRef<HTMLDivElement>(null)

  React.useEffect(() => {
    const host = hostRef.current
    if (!host || reducedMotion || !canUseP5Canvas()) return
    let disposed = false
    let instance: P5RuntimeInstance | null = null
    let onVisibilityChange: (() => void) | null = null

    void loadP5Runtime()
      .then((P5) => {
        if (disposed) return
        instance = new P5(createSwallowSketch(host), host)
        onVisibilityChange = () => {
          if (document.hidden) instance?.noLoop()
          else instance?.loop()
        }
        document.addEventListener("visibilitychange", onVisibilityChange)
      })
      .catch(() => undefined)

    return () => {
      disposed = true
      if (onVisibilityChange) document.removeEventListener("visibilitychange", onVisibilityChange)
      instance?.remove()
    }
  }, [reducedMotion])

  return <div ref={hostRef} aria-hidden="true" className="pointer-events-none absolute inset-0 -z-10 size-full select-none" />
}

interface GrowingTreeDisplay {
  flowerScale: number
  progress: number
}

function createGrowingBranchSketch(
  host: HTMLDivElement,
  displayRef: React.MutableRefObject<GrowingTreeDisplay>,
) {
  return (p: P5Instance) => {
    const branchColor = [145, 132, 132]
    const flowerColor = [224, 137, 164]
    const flowerCenterColor = [247, 195, 104]
    const primaryBranchCount = 3
    const maxLevel = 4
    const growthSpeed = 2.2
    const terminalFlowerProbability = 0.22
    const upwardAngle = -p.HALF_PI
    const leftmostUpwardAngle = -2.85
    const rightmostUpwardAngle = -0.3
    const growRevealEase = 0.008
    const decayRevealEase = 0.06
    let currentFlowerScale = 0.35
    let currentProgress = 0
    let branches: Branch[] = []
    let flowers: Flower[] = []

    interface Point {
      x: number
      y: number
    }

    interface Branch {
      level: number
      revealDuration: number
      revealStart: number
      segments: Point[]
    }

    interface Flower {
      revealAt: number
      size: number
      x: number
      y: number
    }

    const createBranch = (
      x: number,
      y: number,
      level: number,
      angle: number,
      gravitySide: -1 | 0 | 1,
      revealStart: number,
    ) => {
      const maxSegments = Math.max(18, Math.floor(360 * 0.66 ** (level - 1)))
      const maxBuds = level === 1 ? 3 : level === 2 ? 2 : 1
      const branch: Branch = {
        level,
        revealDuration: level === 1 ? 0.28 : 0.22,
        revealStart,
        segments: [{ x, y }],
      }
      const childBranches: Array<{
        angle: number
        gravitySide: -1 | 1
        progress: number
        x: number
        y: number
      }> = []
      let branchAngle = p.constrain(angle + p.random(-0.06, 0.06), leftmostUpwardAngle, rightmostUpwardAngle)
      let lastBudSide: -1 | 1 = p.random() < 0.5 ? -1 : 1
      let nextBudAfter = Math.max(
        8,
        Math.floor(maxSegments * (level === 1 ? p.random(0.18, 0.3) : p.random(0.28, 0.42))),
      )

      for (let index = 1; index <= maxSegments; index += 1) {
        const progress = index / maxSegments
        const gravityTilt = gravitySide * Math.min(0.86, 0.05 + (level - 1) * 0.14 + progress * 0.16)
        const gravityTarget = upwardAngle + gravityTilt
        const upwardPull = 0.012 + Math.max(0, level - 1) * 0.014 + progress * 0.008
        branchAngle = p.constrain(
          branchAngle + p.random(-0.08, 0.08) + (gravityTarget - branchAngle) * upwardPull,
          leftmostUpwardAngle,
          rightmostUpwardAngle,
        )
        const tip = branch.segments[branch.segments.length - 1]
        const next = {
          x: tip.x + p.cos(branchAngle) * growthSpeed,
          y: tip.y + p.sin(branchAngle) * growthSpeed,
        }
        if (next.x < 10 || next.x > p.width - 10 || next.y < 8) break

        branch.segments.push(next)

        if (index % 4 === 0 && p.random() < 0.025 + progress * 0.08) {
          flowers.push({
            revealAt: p.constrain(revealStart + 0.08 + progress * 0.34, 0.08, 0.98),
            size: p.random(8, 14),
            x: next.x,
            y: next.y,
          })
        }

        if (level < maxLevel && childBranches.length < maxBuds && index >= nextBudAfter) {
          const side = (p.random() < 0.25 ? lastBudSide : -lastBudSide) as -1 | 1
          const splitAngle = Math.min(1.05, 0.42 + (level - 1) * 0.06 + progress * 0.18)
          const childAngle = branchAngle + side * splitAngle
          childBranches.push({
            angle: childAngle,
            gravitySide: childAngle < upwardAngle ? -1 : 1,
            progress,
            x: next.x,
            y: next.y,
          })
          lastBudSide = side
          nextBudAfter += Math.max(8, Math.floor(maxSegments * p.lerp(0.3, 0.16, progress)))
        }
      }

      branches.push(branch)
      const tip = branch.segments[branch.segments.length - 1]
      if (branch.segments.length === maxSegments + 1 && p.random() < terminalFlowerProbability) {
        flowers.push({
          revealAt: p.constrain(revealStart + branch.revealDuration, 0.08, 0.98),
          size: p.random(10, 15),
          x: tip.x,
          y: tip.y,
        })
      }

      for (const child of childBranches) {
        const childRevealStart = p.constrain(
          revealStart + 0.12 + child.progress * 0.2 + p.random(-0.03, 0.03),
          0.15,
          0.86,
        )
        createBranch(child.x, child.y, level + 1, child.angle, child.gravitySide, childRevealStart)
      }
    }

    const buildTree = () => {
      branches = []
      flowers = []
      //p.randomSeed(7)
      for (let index = 0; index < primaryBranchCount; index += 1) {
        createBranch(p.width * 0.5, p.height + 18, 1, upwardAngle + p.random(-0.25, 0.25), 0, 0.02)
      }
    }

    const drawFlowers = () => {
      p.noStroke()
      for (const flower of flowers) {
        const visibility = p.constrain((currentProgress - flower.revealAt) / 0.12, 0, 1)
        if (visibility <= 0) continue
        const size = flower.size * currentFlowerScale * (0.7 + visibility * 0.3)
        const opacity = 216 * visibility

        for (let petal = 0; petal < 5; petal += 1) {
          p.push()
          p.translate(flower.x, flower.y)
          p.rotate((p.TWO_PI * petal) / 5)
          p.fill(...flowerColor, opacity)
          p.ellipse(0, -size * 0.32, size * 0.5, size * 0.72)
          p.pop()
        }

        p.fill(...flowerCenterColor, opacity)
        p.ellipse(flower.x, flower.y, size * 0.28, size * 0.28)
      }
    }

    const drawBranches = () => {
      for (const branch of branches) {
        const visibility = p.constrain(
          (currentProgress - branch.revealStart) / branch.revealDuration,
          0,
          1,
        )
        const visibleSegments = Math.floor(branch.segments.length * visibility)
        if (visibleSegments < 2) continue
        p.noFill()
        p.stroke(...branchColor, 120)
        p.strokeCap(p.ROUND)
        p.strokeJoin(p.ROUND)
        for (let index = 1; index < visibleSegments; index += 1) {
          const previous = branch.segments[index - 1]
          const segment = branch.segments[index]
          const taper = 1 - (index / branch.segments.length) * 0.48
          p.strokeWeight(Math.max(0.45, (2.4 / branch.level) * taper))
          p.line(previous.x, previous.y, segment.x, segment.y)
        }
      }
    }

    p.setup = () => {
      const bounds = host.getBoundingClientRect()
      p.pixelDensity(Math.min(window.devicePixelRatio || 1, 2))
      p.createCanvas(Math.max(1, bounds.width), Math.max(1, bounds.height))
      p.canvas.setAttribute("aria-hidden", "true")
      p.canvas.className = "size-full"
      buildTree()
    }

    p.draw = () => {
      p.clear()
      if (document.hidden) return
      const target = displayRef.current
      const progressEase = target.progress < currentProgress ? decayRevealEase : growRevealEase
      const flowerEase = target.flowerScale < currentFlowerScale ? decayRevealEase : growRevealEase
      currentProgress = p.lerp(currentProgress, target.progress, progressEase)
      currentFlowerScale = p.lerp(currentFlowerScale, target.flowerScale, flowerEase)
      drawBranches()
      drawFlowers()
    }

    p.windowResized = () => {
      const bounds = host.getBoundingClientRect()
      p.resizeCanvas(Math.max(1, bounds.width), Math.max(1, bounds.height))
      buildTree()
    }
  }
}

export function SupportGrowingBranch({
  flowerScale = 0.7,
  reducedMotion = false,
  reveal = 0.22,
  trigger,
}: {
  flowerScale?: number
  reducedMotion?: boolean
  reveal?: number
  trigger: number
}) {
  const hostRef = React.useRef<HTMLDivElement>(null)
  const displayRef = React.useRef<GrowingTreeDisplay>({ flowerScale, progress: reveal })
  displayRef.current = { flowerScale, progress: reveal }

  React.useEffect(() => {
    const host = hostRef.current
    if (!host || reducedMotion || trigger === 0 || !canUseP5Canvas()) return
    let disposed = false
    let instance: P5RuntimeInstance | null = null
    let onVisibilityChange: (() => void) | null = null

    void loadP5Runtime()
      .then((P5) => {
        if (disposed) return
        instance = new P5(createGrowingBranchSketch(host, displayRef), host)
        onVisibilityChange = () => {
          if (document.hidden) instance?.noLoop()
          else instance?.loop()
        }
        document.addEventListener("visibilitychange", onVisibilityChange)
      })
      .catch(() => undefined)

    return () => {
      disposed = true
      if (onVisibilityChange) document.removeEventListener("visibilitychange", onVisibilityChange)
      instance?.remove()
    }
  }, [reducedMotion, trigger])

  return <div ref={hostRef} aria-hidden="true" className="pointer-events-none fixed inset-x-0 bottom-0 -z-10 h-[74vh] select-none md:h-[78vh]" />
}
