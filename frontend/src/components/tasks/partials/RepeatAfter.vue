<template>
	<div class="control repeat-after-input">
		<RecurrenceSeriesEditor
			v-if="task"
			:task-id="task.id"
			:due-date="task.dueDate"
			:recurrence="task.recurrence"
			:disabled="disabled"
		/>
	</div>
</template>

<script setup lang="ts">
import {computed} from 'vue'

import RecurrenceSeriesEditor from '@/components/tasks/partials/RecurrenceSeriesEditor.vue'

import type {ITask} from '@/modelTypes/ITask'
const props = withDefaults(defineProps<{
	modelValue: ITask | undefined
	disabled?: boolean
}>(), {
	disabled: false,
})

const task = computed(() => props.modelValue)

</script>

<style scoped>
.repeat-after-input {
	width: 100%;
}
</style>
