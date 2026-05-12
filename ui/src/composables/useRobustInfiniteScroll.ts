import { computed, nextTick, ref, toValue, watch, type MaybeRefOrGetter, type Ref } from 'vue';
import { useEventListener, useIntersectionObserver, useResizeObserver } from '@vueuse/core';

export interface UseRobustInfiniteScrollOptions {
  containerRef: Ref<HTMLElement | null>;
  sentinelRef: Ref<HTMLElement | null>;
  canLoadMore: MaybeRefOrGetter<boolean>;
  onLoadMore: () => void | Promise<void>;
  enabled?: MaybeRefOrGetter<boolean>;
  loading?: MaybeRefOrGetter<boolean>;
  triggerDeps?: () => unknown[];
  bottomOffset?: number;
  rootMargin?: string;
  threshold?: number;
  scrollFallback?: boolean;
  observeResize?: boolean;
  requestAnimationFrameCheck?: boolean;
  observeMutations?: boolean;
}

const nextFrame = async () => {
  if (typeof window === 'undefined' || typeof requestAnimationFrame !== 'function') {
    return;
  }
  await new Promise<void>((resolve) => {
    requestAnimationFrame(() => resolve());
  });
};

const isElementNearBottom = (container: HTMLElement, bottomOffset: number) =>
  container.scrollTop + container.clientHeight >= container.scrollHeight - bottomOffset;

const isElementShortForOverflow = (container: HTMLElement, bottomOffset: number) =>
  container.scrollHeight <= container.clientHeight + bottomOffset;

export function useRobustInfiniteScroll(options: UseRobustInfiniteScrollOptions) {
  const {
    containerRef,
    sentinelRef,
    canLoadMore,
    onLoadMore,
    triggerDeps,
    bottomOffset = 40,
    rootMargin = '0px 0px 80px 0px',
    threshold = 0.01,
    scrollFallback = true,
    observeResize = true,
    requestAnimationFrameCheck = true,
    observeMutations = true,
  } = options;

  const enabled = options.enabled ?? true;
  const loading = options.loading ?? false;

  const isEnabled = computed(() => toValue(enabled) !== false);
  const isLoadBlocked = computed(() => !isEnabled.value || !toValue(canLoadMore) || !!toValue(loading));
  const loadPending = ref(false);

  const shouldLoadMore = (container: HTMLElement, force = false) => {
    if (isLoadBlocked.value) {
      return false;
    }
    return force || isElementShortForOverflow(container, bottomOffset) || isElementNearBottom(container, bottomOffset);
  };

  const runLoadMore = async () => {
    if (loadPending.value || isLoadBlocked.value) {
      return;
    }
    loadPending.value = true;
    try {
      await onLoadMore();
      await nextTick();
      if (requestAnimationFrameCheck) {
        await nextFrame();
      }
    } finally {
      loadPending.value = false;
    }
  };

  const checkAndLoadMore = async (force = false) => {
    const container = containerRef.value;
    if (!container || !shouldLoadMore(container, force)) {
      return;
    }
    await runLoadMore();
  };

  const scheduleCheck = (force = false) => {
    void checkAndLoadMore(force);
  };

  useIntersectionObserver(
    sentinelRef,
    ([entry]) => {
      if (!entry?.isIntersecting) {
        return;
      }
      scheduleCheck(true);
    },
    {
      root: containerRef,
      rootMargin,
      threshold,
    }
  );

  if (scrollFallback) {
    useEventListener(
      containerRef,
      'scroll',
      () => {
        scheduleCheck();
      },
      { passive: true }
    );
  }

  if (observeResize) {
    useResizeObserver(containerRef, () => {
      scheduleCheck();
    });

    useEventListener(
      containerRef,
      'load',
      () => {
        scheduleCheck();
      },
      { passive: true, capture: true }
    );
  }

  let mutationObserver: MutationObserver | null = null;

  watch(
    containerRef,
    (container, previous) => {
      if (mutationObserver && previous) {
        mutationObserver.disconnect();
        mutationObserver = null;
      }
      if (!observeMutations || !container || typeof MutationObserver === 'undefined') {
        return;
      }
      mutationObserver = new MutationObserver(() => {
        scheduleCheck();
      });
      mutationObserver.observe(container, {
        childList: true,
        subtree: true,
      });
    },
    { immediate: true }
  );

  watch(
    () => [
      isEnabled.value,
      toValue(canLoadMore),
      !!toValue(loading),
      ...(triggerDeps?.() ?? []),
    ],
    async () => {
      if (toValue(loading)) {
        loadPending.value = false;
        return;
      }
      await nextTick();
      scheduleCheck();
    },
    { immediate: true }
  );

  return {
    checkNow: () => checkAndLoadMore(true),
  };
}
