<script lang="ts">
  import { onMount } from 'svelte'
  import Icon from '@iconify/svelte'
  import * as Dialog from '$lib/components/ui/dialog'
  import { Button } from '$lib/components/ui/button'
  import { Input } from '$lib/components/ui/input'
  import { Label } from '$lib/components/ui/label'
  import * as Select from '$lib/components/ui/select'
  import { _ } from '$lib/i18n'
  import { format, addHours, startOfHour, parseISO } from 'date-fns'
  // @ts-ignore - wailsjs
  import { UpsertCalendarEvent } from '../../../../wailsjs/go/app/App'
  // @ts-ignore - wailsjs models
  import { calendar } from '../../../../wailsjs/go/models'

  interface Props {
    open?: boolean
    event?: calendar.Event | null
    initialDate?: Date
    calendars: calendar.Calendar[]
    onSave?: () => void
    onClose?: () => void
  }

  let {
    open = $bindable(false),
    event = null,
    initialDate = new Date(),
    calendars,
    onSave,
    onClose,
  }: Props = $props()

  let summary = $state('')
  let description = $state('')
  let location = $state('')
  let startDate = $state('')
  let startTime = $state('')
  let endDate = $state('')
  let endTime = $state('')
  let isAllDay = $state(false)
  let calendarId = $state('')
  let saving = $state(false)

  const writableCalendars = $derived(calendars.filter(c => c.accessRole === 'writer' || c.accessRole === 'owner'))

  $effect(() => {
    if (open) {
      if (event) {
        summary = event.summary
        description = event.description || ''
        location = event.location || ''
        isAllDay = event.isAllDay
        calendarId = event.calendarId
        
        const start = new Date(event.startTime)
        const end = new Date(event.endTime)
        
        startDate = format(start, 'yyyy-MM-dd')
        startTime = format(start, 'HH:mm')
        endDate = format(end, 'yyyy-MM-dd')
        endTime = format(end, 'HH:mm')
      } else {
        summary = ''
        description = ''
        location = ''
        isAllDay = false
        
        // Default to first enabled calendar that is also writable
        const defaultCal = writableCalendars.find(c => c.enabled) || writableCalendars[0]
        calendarId = defaultCal?.id || ''

        const start = initialDate ? startOfHour(initialDate) : startOfHour(new Date())
        const end = addHours(start, 1)

        startDate = format(start, 'yyyy-MM-dd')
        startTime = format(start, 'HH:mm')
        endDate = format(end, 'yyyy-MM-dd')
        endTime = format(end, 'HH:mm')
      }
    }
  })

  async function handleSave() {
    if (!summary || !calendarId) {
      return
    }
    
    saving = true
    try {
      const start = parseISO(`${startDate}T${isAllDay ? '00:00' : startTime}`)
      const end = parseISO(`${endDate}T${isAllDay ? '23:59' : endTime}`)

      const newEvent = new calendar.Event({
        id: event?.id || '',
        calendarId,
        summary,
        description,
        location,
        startTime: start.toISOString(),
        endTime: end.toISOString(),
        isAllDay,
        status: 'confirmed',
      })

      await UpsertCalendarEvent(newEvent)
      open = false
      onSave?.()
    } catch (err) {
      console.error('Failed to save event:', err)
    } finally {
      saving = false
    }
  }

  const selectedCalendar = $derived(calendars.find(c => c.id === calendarId))
</script>

<Dialog.Root bind:open onOpenChange={(isOpen) => !isOpen && onClose?.()}>
  <Dialog.Content class="sm:max-w-[425px]">
    <Dialog.Header>
      <Dialog.Title>{event ? $_('calendar.editEvent') : $_('calendar.newEvent')}</Dialog.Title>
    </Dialog.Header>

    <div class="grid gap-4 py-4">
      <div class="grid gap-2">
        <Label for="summary">{$_('calendar.summary')}</Label>
        <Input id="summary" bind:value={summary} placeholder="Event title" />
      </div>

      <div class="grid grid-cols-2 gap-4">
        <div class="grid gap-2">
          <Label for="startDate">{$_('calendar.start')}</Label>
          <Input id="startDate" type="date" bind:value={startDate} />
          {#if !isAllDay}
            <Input type="time" bind:value={startTime} />
          {/if}
        </div>
        <div class="grid gap-2">
          <Label for="endDate">{$_('calendar.end')}</Label>
          <Input id="endDate" type="date" bind:value={endDate} />
          {#if !isAllDay}
            <Input type="time" bind:value={endTime} />
          {/if}
        </div>
      </div>

      <div class="flex items-center space-x-2">
        <input type="checkbox" id="isAllDay" bind:checked={isAllDay} class="h-4 w-4 rounded border-gray-300 text-primary focus:ring-primary" />
        <Label for="isAllDay">{$_('calendar.allDay')}</Label>
      </div>

      <div class="grid gap-2">
        <Label for="calendar">{$_('calendar.calendar')}</Label>
        <Select.Root bind:value={calendarId}>
          <Select.Trigger class="w-full">
            {#if selectedCalendar}
              <div class="flex items-center gap-2">
                <div class="w-3 h-3 rounded-full" style="background-color: {selectedCalendar.color}"></div>
                <span>{selectedCalendar.name}</span>
              </div>
            {:else}
              <span>Select calendar</span>
            {/if}
          </Select.Trigger>
          <Select.Content>
            {#each writableCalendars as cal}
              <Select.Item value={cal.id} label={cal.name}>
                <div class="flex items-center gap-2">
                  <div class="w-3 h-3 rounded-full" style="background-color: {cal.color}"></div>
                  <span>{cal.name}</span>
                </div>
              </Select.Item>
            {/each}
          </Select.Content>
        </Select.Root>
      </div>

      <div class="grid gap-2">
        <Label for="location">{$_('calendar.location')}</Label>
        <Input id="location" bind:value={location} placeholder="Add location" />
      </div>

      <div class="grid gap-2">
        <Label for="description">{$_('calendar.description')}</Label>
        <textarea
          id="description"
          bind:value={description}
          class="flex min-h-[80px] w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
          placeholder="Add description"
        ></textarea>
      </div>
    </div>

    <Dialog.Footer>
      <Button variant="outline" onclick={() => open = false}>{$_('common.cancel')}</Button>
      <Button onclick={handleSave} disabled={saving || !summary || !calendarId}>
        {#if saving}
          <Icon icon="mdi:loading" class="w-4 h-4 mr-2 animate-spin" />
        {/if}
        {$_('common.save')}
      </Button>
    </Dialog.Footer>
  </Dialog.Content>
</Dialog.Root>
