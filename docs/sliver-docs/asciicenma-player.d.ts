declare module "asciinema-player" {
  export function create(
    src: string,
    element: HTMLElement | null,
    // START asciinemaOptions
    opts: {
      cols?: string,
      rows?: string,
      autoPlay?: boolean,
      preload?: boolean,
      loop?: boolean | number,
      startAt?: number | string,
      speed?: number,
      idleTimeLimit?: number,
      theme?: string,
      poster?: string,
      fit?: string,
      fontSize?: string
    }
  )
}