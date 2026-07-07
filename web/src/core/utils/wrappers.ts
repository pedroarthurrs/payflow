import { createTestingPinia } from '@pinia/testing'
import { QueryClient, VueQueryPlugin } from '@tanstack/vue-query'
import { mount, type ComponentMountingOptions } from '@vue/test-utils'
import type { Component } from 'vue'

/**
 * Mounts a component with TanStack Query and Pinia pre-configured for testing.
 *
 * - QueryClient uses `retry: false` and `gcTime: 0` for test isolation.
 * - Pinia is provided via `createTestingPinia()` with stubbed actions.
 */
export function mountWithQuery<T extends Component>(
    component: T,
    options: ComponentMountingOptions<T> = {}
) {
    const queryClient = new QueryClient({
        defaultOptions: {
            queries: {
                retry: false,
                gcTime: 0,
            },
        },
    })

    const { global: globalOptions, ...restOptions } = options

    return mount(component, {
        global: {
            plugins: [
                [VueQueryPlugin, { queryClient }],
                createTestingPinia(),
                ...(globalOptions?.plugins ?? []),
            ],
            ...globalOptions,
        },
        ...restOptions,
    } as ComponentMountingOptions<T>)
}
