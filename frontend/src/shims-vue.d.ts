declare module "*.vue" {
  import type { DefineComponent } from "vue";
  const component: DefineComponent<{}, {}, any>;
  export default component;
}

declare module "plotly.js-dist-min" {
  export interface Data {
    x?: number[];
    y?: number[];
    mode?: string;
    type?: string;
    name?: string;
    hovertemplate?: string;
    line?: { color?: string; width?: number };
  }

  export interface Layout {
    paper_bgcolor?: string;
    plot_bgcolor?: string;
    font?: { color?: string; family?: string; size?: number };
    margin?: { l?: number; r?: number; t?: number; b?: number };
    xaxis?: {
      title?: { text?: string };
      gridcolor?: string;
      zerolinecolor?: string;
      showspikes?: boolean;
      spikemode?: string;
      spikethickness?: number;
      spikecolor?: string;
    };
    yaxis?: {
      title?: { text?: string };
      gridcolor?: string;
      zerolinecolor?: string;
    };
    legend?: { orientation?: string; x?: number; y?: number; bgcolor?: string };
    hovermode?: string;
    dragmode?: string;
  }

  export interface Config {
    displaylogo?: boolean;
    responsive?: boolean;
    scrollZoom?: boolean;
    toImageButtonOptions?: {
      format?: string;
      filename?: string;
      height?: number;
      width?: number;
      scale?: number;
    };
    modeBarButtonsToAdd?: string[];
    modeBarButtonsToRemove?: string[];
  }

  export function newPlot(
    element: HTMLElement,
    data: Data[],
    layout: Partial<Layout>,
    config?: Partial<Config>
  ): Promise<void>;

  export function react(
    element: HTMLElement,
    data: Data[],
    layout: Partial<Layout>,
    config?: Partial<Config>
  ): Promise<void>;

  export function purge(element: HTMLElement): void;

  export namespace Plots {
    export function resize(element: HTMLElement): void;
  }
}
