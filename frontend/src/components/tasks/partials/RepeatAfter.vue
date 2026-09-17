<template>
	<div class="control repeat-after-input">
		<RecurrenceSeriesEditor
			v-if="task"
			:task-id="task.id"
			:due-date="task.dueDate"
			:recurrence="task.recurrence"
			:disabled="disabled"
			@update:recurrence="updateRecurrence"
		/>
	</div>
</template>

<script setup lang="ts">
import {computed} from 'vue'

import RecurrenceSeriesEditor from '@/components/tasks/partials/RecurrenceSeriesEditor.vue'

import type {ITask} from '@/modelTypes/ITask'
import type {ITaskRecurrence} from '@/modelTypes/ITaskRecurrence'
import TaskModel from '@/models/task'

const props = withDefaults(defineProps<{
	modelValue: ITask | undefined
	disabled?: boolean
}>(), {
	disabled: false,
})

const emit = defineEmits<{
	'update:modelValue': [value: ITask | undefined]
}>()

const task = computed(() => props.modelValue)

function updateRecurrence(recurrence: ITaskRecurrence | null) {
	if (!props.modelValue) {
		return
	}

	const updated = new TaskModel(props.modelValue)
	updated.recurrence = recurrence

	emit('update:modelValue', updated)
}
</script>

<style scoped>
.repeat-after-input {
	width: 100%;
}
</style>
