<script setup lang="ts">
import { cn } from "@/core/utils/formatting";
import { reactiveOmit } from "@vueuse/core";
import type { SelectSeparatorProps } from "reka-ui";
import { SelectSeparator } from "reka-ui";
import type { HTMLAttributes } from "vue";

const props = defineProps<SelectSeparatorProps & { class?: HTMLAttributes["class"] }>()

const delegatedProps = reactiveOmit(props, "class")
</script>

<template>
  <SelectSeparator v-bind="delegatedProps" :class="cn('-mx-1 my-1 h-px bg-muted', props.class)" />
</template>
