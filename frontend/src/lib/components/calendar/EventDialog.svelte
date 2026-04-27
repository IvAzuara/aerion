<script lang="ts">
  import { onMount } from 'svelte'
  import Icon from '@iconify/svelte'
  import * as Dialog from '$lib/components/ui/dialog'
  import { Button } from '$lib/components/ui/button'
  import { Input } from '$lib/components/ui/input'
  import { Label } from '$lib/components/ui/label'
  import * as Select from '$lib/components/ui/select'
  import * as DropdownMenu from '$lib/components/ui/dropdown-menu'
  import TimePicker from './TimePicker.svelte'
  import { _ } from '$lib/i18n'
  import { 
    format, 
    addHours, 
    startOfHour, 
    parseISO, 
    startOfMonth, 
    endOfMonth, 
    startOfWeek, 
    endOfWeek, 
    eachDayOfInterval, 
    isSameDay, 
    addMonths, 
    subMonths,
    isSameMonth,
    addDays,
    parse,
    isBefore,
    differenceInMinutes
  } from 'date-fns'
  import { generateTimeOptions } from '$lib/utils/time'
  import { ThreeOptionDialog } from '$lib/components/ui/confirm-dialog'
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
  let startDate = $state(format(new Date(), 'yyyy-MM-dd'))
  let startTime = $state(format(new Date(), 'HH:mm'))
  let endDate = $state(format(new Date(), 'yyyy-MM-dd'))
  let endTime = $state(format(new Date(), 'HH:mm'))
  let isAllDay = $state(false)
  let recurrence = $state('')
  let calendarId = $state('')
  let createMeetLink = $state(false)
  let saving = $state(false)
  let attendees = $state<string[]>([])
  let newAttendee = $state('')
  let showInviteConfirm = $state(false)

  const allTimeOptions = generateTimeOptions()

  // Recurrence options
  const getRecurrenceOptions = () => {
    const start = parseISO(`${startDate}T${startTime}`)
    const dayName = format(start, 'EEEE')
    const dayShort = format(start, 'EEEEEE').toUpperCase()
    const monthDay = format(start, 'd')
    const ordinal = Math.ceil(parseInt(monthDay) / 7)
    const ordinalName = ['first', 'second', 'third', 'fourth', 'fifth'][ordinal - 1]

    return [
      { label: $_('calendar.recurrence.none') || 'Does not repeat', value: '' },
      { label: $_('calendar.recurrence.daily') || 'Daily', value: 'RRULE:FREQ=DAILY' },
      { label: ($_('calendar.recurrence.weekly') || 'Weekly on ') + dayName, value: `RRULE:FREQ=WEEKLY;BYDAY=${dayShort}` },
      { label: ($_('calendar.recurrence.monthly') || 'Monthly on the ') + ordinalName + ' ' + dayName, value: `RRULE:FREQ=MONTHLY;BYDAY=${ordinal}${dayShort}` },
      { label: ($_('calendar.recurrence.annually') || 'Annually on ') + format(start, 'MMMM d'), value: `RRULE:FREQ=YEARLY` },
      { label: $_('calendar.recurrence.weekdays') || 'Every weekday (Monday to Friday)', value: 'RRULE:FREQ=WEEKLY;BYDAY=MO,TU,WE,TH,FR' },
    ]
  }

  const recurrenceOptions = $derived(getRecurrenceOptions())

  // Derived options for end time to only show times after start time if on same day
  const endTimeOptions = $derived.by(() => {
    if (startDate === endDate) {
      return allTimeOptions.filter(t => t > startTime)
    }
    return allTimeOptions
  })

  // Watchers to ensure end is after start
  $effect(() => {
    // If start date/time changes, ensure end is at least start + 1 hour (or same if user manually adjusted)
    const start = parseISO(`${startDate}T${startTime}`)
    const end = parseISO(`${endDate}T${endTime}`)
    
    if (isBefore(end, start)) {
      // If same day but end time is before start time, move end date to next day (Google style)
      if (startDate === endDate) {
        endDate = format(addDays(start, 1), 'yyyy-MM-dd')
      } else if (isBefore(parseISO(endDate), parseISO(startDate))) {
        endDate = startDate
      }
    }
  })

  // Custom picker state
  let viewDateStart = $state(new Date())
  let viewDateEnd = $state(new Date())

  function safeFormat(dateStr: string, fmt: string) {
    try {
      if (!dateStr) return ''
      const d = parseISO(dateStr)
      if (isNaN(d.getTime())) return ''
      return format(d, fmt)
    } catch (e) {
      return ''
    }
  }

  function safeFormatDate(date: Date, fmt: string) {
    try {
      if (!date || isNaN(date.getTime())) return ''
      return format(date, fmt)
    } catch (e) {
      return ''
    }
  }
  
  function getDaysInMonth(date: Date) {
    const start = startOfWeek(startOfMonth(date))
    const end = endOfWeek(endOfMonth(date))
    return eachDayOfInterval({ start, end })
  }

  function selectDate(date: Date, type: 'start' | 'end') {
    if (type === 'start') {
      const oldStart = parseISO(startDate)
      
      startDate = format(date, 'yyyy-MM-dd')
      
      // Move end date by same amount to maintain duration (Google style)
      const newEnd = addDays(parseISO(endDate), differenceInMinutes(date, oldStart) / (24 * 60))
      endDate = format(date > parseISO(endDate) ? date : newEnd, 'yyyy-MM-dd')
    } else {
      const selectedDateStr = format(date, 'yyyy-MM-dd')
      if (selectedDateStr < startDate) {
        endDate = startDate
      } else {
        endDate = selectedDateStr
      }
    }
  }

  const writableCalendars = $derived(calendars.filter(c => c.accessRole === 'writer' || c.accessRole === 'owner'))

  $effect(() => {
    if (open) {
      if (event) {
        summary = event.summary
        description = event.description || ''
        location = event.location || ''
        isAllDay = event.isAllDay
        calendarId = event.calendarId
        recurrence = event.recurrence || ''
        createMeetLink = false
        attendees = event.attendees?.map((a: any) => a.email) || []
        
        const start = new Date(event.startTime)
        const end = new Date(event.endTime)
        
        startDate = format(start, 'yyyy-MM-dd')
        startTime = format(start, 'HH:mm')
        endDate = format(end, 'yyyy-MM-dd')
        endTime = format(end, 'HH:mm')

        viewDateStart = start
        viewDateEnd = end
      } else {
        summary = ''
        description = ''
        location = ''
        isAllDay = false
        recurrence = ''
        attendees = []
        
        // Default to first enabled calendar that is also writable
        const defaultCal = writableCalendars.find(c => c.enabled) || writableCalendars[0]
        calendarId = defaultCal?.id || ''

        const start = initialDate ? startOfHour(initialDate) : startOfHour(new Date())
        const end = addHours(start, 1)

        startDate = format(start, 'yyyy-MM-dd')
        startTime = format(start, 'HH:mm')
        endDate = format(end, 'yyyy-MM-dd')
        endTime = format(end, 'HH:mm')

        viewDateStart = start
        viewDateEnd = end
      }
    }
  })

  function addAttendee() {
    if (newAttendee && !attendees.includes(newAttendee)) {
      attendees = [...attendees, newAttendee]
      newAttendee = ''
    }
  }

  function removeAttendee(email: string) {
    attendees = attendees.filter(a => a !== email)
  }

  async function handleSave() {
    if (!summary || !calendarId) {
      return
    }

    const selectedCal = calendars.find(c => c.id === calendarId)
    if (selectedCal?.type === 'google' && attendees.length > 0) {
      showInviteConfirm = true
      return
    }
    
    await doSave(false)
  }

  async function doSave(sendInvitations: boolean) {
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
        recurrence,
        status: 'confirmed',
        attendees: attendees.map(email => ({ email })),
      })

      // @ts-ignore - wailsjs bindings update sync
      await UpsertCalendarEvent(newEvent, createMeetLink, sendInvitations)
      open = false
      onSave?.()
    } catch (err) {
      console.error('Failed to save event:', err)
    } finally {
      saving = false
      showInviteConfirm = false
    }
  }

  const selectedCalendar = $derived(calendars.find(c => c.id === calendarId))
</script>

<Dialog.Root bind:open onOpenChange={(isOpen) => !isOpen && onClose?.()}>
  <Dialog.Content class="sm:max-w-[500px] max-h-[90vh] overflow-y-auto">
    <Dialog.Header>
      <Dialog.Title>{event ? $_('calendar.editEvent') : $_('calendar.newEvent')}</Dialog.Title>
    </Dialog.Header>

    <div class="grid gap-4 py-4">
      <div class="grid gap-2">
        <Label for="summary">{$_('calendar.summary')}</Label>
        <Input id="summary" bind:value={summary} placeholder="Event title" />
      </div>

      <div class="grid grid-cols-2 gap-4">
        <!-- Start Date/Time -->
        <div class="grid gap-2">
          <Label>{$_('calendar.start')}</Label>
          <div class="flex flex-col gap-2">
            <!-- Start Date Picker -->
            <DropdownMenu.Root>
              <DropdownMenu.Trigger class="w-full">
                <Button 
                  variant="outline" 
                  class="w-full justify-start text-left font-normal"
                >
                  <Icon icon="mdi:calendar" class="mr-2 h-4 w-4 opacity-50" />
                  {safeFormat(startDate, 'PPP')}
                </Button>
              </DropdownMenu.Trigger>
              <DropdownMenu.Content class="w-auto p-0" align="start">
                <div class="p-3 bg-background border rounded-md shadow-md">
                  <div class="flex items-center justify-between mb-4 px-1">
                    <span class="text-sm font-semibold">{safeFormatDate(viewDateStart, 'MMMM yyyy')}</span>
                    <div class="flex items-center gap-1">
                      <Button variant="ghost" size="icon" class="h-7 w-7" onclick={() => viewDateStart = subMonths(viewDateStart, 1)}>
                        <Icon icon="mdi:chevron-left" class="h-4 w-4" />
                      </Button>
                      <Button variant="ghost" size="icon" class="h-7 w-7" onclick={() => viewDateStart = addMonths(viewDateStart, 1)}>
                        <Icon icon="mdi:chevron-right" class="h-4 w-4" />
                      </Button>
                    </div>
                  </div>
                  <div class="grid grid-cols-7 gap-1 text-center mb-2">
                    {#each ['Su', 'Mo', 'Tu', 'We', 'Th', 'Fr', 'Sa'] as day}
                      <span class="text-[10px] font-medium text-muted-foreground">{day}</span>
                    {/each}
                  </div>
                  <div class="grid grid-cols-7 gap-1">
                    {#each getDaysInMonth(viewDateStart) as day}
                      <Button
                        variant="ghost"
                        class="h-8 w-8 p-0 font-normal {isSameDay(day, parseISO(startDate)) ? 'bg-primary text-primary-foreground hover:bg-primary hover:text-primary-foreground' : ''} {!isSameMonth(day, viewDateStart) ? 'opacity-30' : ''}"
                        onclick={() => selectDate(day, 'start')}
                      >
                        <span class="text-xs">{format(day, 'd')}</span>
                      </Button>
                    {/each}
                  </div>
                </div>
              </DropdownMenu.Content>
            </DropdownMenu.Root>

            <!-- Start Time Picker -->
            <TimePicker bind:value={startTime} disabled={isAllDay} />
          </div>
        </div>

        <!-- End Date/Time -->
        <div class="grid gap-2">
          <Label>{$_('calendar.end')}</Label>
          <div class="flex flex-col gap-2">
            <!-- End Date Picker -->
            <DropdownMenu.Root>
              <DropdownMenu.Trigger class="w-full">
                <Button 
                  variant="outline" 
                  class="w-full justify-start text-left font-normal"
                >
                  <Icon icon="mdi:calendar" class="mr-2 h-4 w-4 opacity-50" />
                  {safeFormat(endDate, 'PPP')}
                </Button>
              </DropdownMenu.Trigger>
              <DropdownMenu.Content class="w-auto p-0" align="start">
                <div class="p-3 bg-background border rounded-md shadow-md">
                  <div class="flex items-center justify-between mb-4 px-1">
                    <span class="text-sm font-semibold">{safeFormatDate(viewDateEnd, 'MMMM yyyy')}</span>
                    <div class="flex items-center gap-1">
                      <Button variant="ghost" size="icon" class="h-7 w-7" onclick={() => viewDateEnd = subMonths(viewDateEnd, 1)}>
                        <Icon icon="mdi:chevron-left" class="h-4 w-4" />
                      </Button>
                      <Button variant="ghost" size="icon" class="h-7 w-7" onclick={() => viewDateEnd = addMonths(viewDateEnd, 1)}>
                        <Icon icon="mdi:chevron-right" class="h-4 w-4" />
                      </Button>
                    </div>
                  </div>
                  <div class="grid grid-cols-7 gap-1 text-center mb-2">
                    {#each ['Su', 'Mo', 'Tu', 'We', 'Th', 'Fr', 'Sa'] as day}
                      <span class="text-[10px] font-medium text-muted-foreground">{day}</span>
                    {/each}
                  </div>
                  <div class="grid grid-cols-7 gap-1">
                    {#each getDaysInMonth(viewDateEnd) as day}
                      <Button
                        variant="ghost"
                        class="h-8 w-8 p-0 font-normal {isSameDay(day, parseISO(endDate)) ? 'bg-primary text-primary-foreground hover:bg-primary hover:text-primary-foreground' : ''} {!isSameMonth(day, viewDateEnd) ? 'opacity-30' : ''}"
                        onclick={() => selectDate(day, 'end')}
                      >
                        <span class="text-xs">{format(day, 'd')}</span>
                      </Button>
                    {/each}
                  </div>
                </div>
              </DropdownMenu.Content>
            </DropdownMenu.Root>

            <!-- End Time Picker -->
            <TimePicker bind:value={endTime} disabled={isAllDay} options={endTimeOptions} />
          </div>
        </div>
      </div>

      <div class="flex items-center gap-4">
        <button 
          type="button"
          class="flex items-center gap-3 p-3 flex-1 rounded-xl border transition-all duration-200 group/allday {isAllDay ? 'bg-primary/5 border-primary/30' : 'bg-muted/20 hover:bg-muted/30 border-transparent'}"
          onclick={() => isAllDay = !isAllDay}
        >
          <div class="flex items-center justify-center w-5 h-5 rounded border transition-all duration-200 {isAllDay ? 'bg-primary border-primary' : 'border-muted-foreground/30'}">
            {#if isAllDay}
              <Icon icon="mdi:check" class="w-3.5 h-3.5 text-white" />
            {/if}
          </div>
          <span class="text-sm font-medium transition-colors {isAllDay ? 'text-foreground' : 'text-muted-foreground'}">
            {$_('calendar.allDay')}
          </span>
        </button>

        <div class="flex-1">
          <Select.Root bind:value={recurrence}>
            <Select.Trigger class="w-full">
              <div class="flex items-center gap-2">
                <Icon icon="mdi:repeat" class="w-4 h-4 opacity-50" />
                <span>{recurrenceOptions.find(o => o.value === recurrence)?.label || recurrenceOptions[0].label}</span>
              </div>
            </Select.Trigger>
            <Select.Content>
              {#each recurrenceOptions as opt}
                <Select.Item value={opt.value} label={opt.label}>
                  {opt.label}
                </Select.Item>
              {/each}
            </Select.Content>
          </Select.Root>
        </div>
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

      {#if selectedCalendar?.type === 'google'}
        <div class="grid gap-2 p-4 rounded-xl border bg-muted/20 transition-all duration-200">
          {#if event?.meetLink}
            <div class="flex items-center justify-between">
              <div class="flex items-center gap-3 text-sm">
                <div class="p-2 rounded-full bg-primary/10">
                  <Icon icon="mdi:video" class="w-5 h-5 text-primary" />
                </div>
                <div>
                  <span class="block font-semibold text-foreground">{$_('calendar.videoCall') || 'Video call'}</span>
                  <span class="text-[10px] text-muted-foreground">Google Meet</span>
                </div>
              </div>
              <a 
                href={event.meetLink} 
                target="_blank" 
                rel="noopener noreferrer" 
                class="px-4 py-1.5 rounded-full text-xs font-medium bg-primary text-primary-foreground hover:bg-primary/90 transition-colors shadow-sm"
              >
                {$_('calendar.join') || 'Join'}
              </a>
            </div>
          {:else}
            <button 
              type="button"
              class="flex items-center justify-between w-full text-left transition-all duration-200 group/meet"
              onclick={() => createMeetLink = !createMeetLink}
            >
              <div class="flex items-center gap-3">
                <div class="p-2 rounded-full transition-colors {createMeetLink ? 'bg-primary/10' : 'bg-muted-foreground/10 group-hover/meet:bg-muted-foreground/20'}">
                  <Icon 
                    icon="mdi:video-plus" 
                    class="w-5 h-5 transition-colors {createMeetLink ? 'text-primary' : 'text-muted-foreground'}" 
                  />
                </div>
                <div>
                  <span class="block text-sm font-semibold transition-colors {createMeetLink ? 'text-foreground' : 'text-muted-foreground'}">
                    {$_('calendar.addMeet') || 'Add Google Meet'}
                  </span>
                  <span class="text-[10px] text-muted-foreground">
                    {$_('calendar.meetDescription') || 'Generate a link for this event'}
                  </span>
                </div>
              </div>
              
              <div class="flex items-center justify-center w-6 h-6 rounded-full border-2 transition-all duration-200 {createMeetLink ? 'bg-green-500 border-green-500 scale-100' : 'border-muted-foreground/30 scale-90'}">
                {#if createMeetLink}
                  <Icon icon="mdi:check" class="w-4 h-4 text-white" />
                {/if}
              </div>
            </button>
          {/if}
        </div>
      {/if}

      <div class="grid gap-2">
        <Label for="location">{$_('calendar.location')}</Label>
        <Input id="location" bind:value={location} placeholder="Add location" />
      </div>

      <div class="grid gap-2">
        <Label>{$_('calendar.guests') || 'Guests'}</Label>
        <div class="flex gap-2">
          <Input 
            placeholder="Add guest email" 
            bind:value={newAttendee} 
            onkeydown={(e) => e.key === 'Enter' && (e.preventDefault(), addAttendee())}
          />
          <Button variant="outline" size="icon" onclick={addAttendee}>
            <Icon icon="mdi:plus" class="w-4 h-4" />
          </Button>
        </div>
        {#if attendees.length > 0}
          <div class="flex flex-wrap gap-2 mt-2">
            {#each attendees as email}
              <div class="flex items-center gap-1 px-2 py-1 rounded-full bg-secondary text-secondary-foreground text-xs">
                <span>{email}</span>
                <button onclick={() => removeAttendee(email)} class="hover:text-destructive">
                  <Icon icon="mdi:close" class="w-3 h-3" />
                </button>
              </div>
            {/each}
          </div>
        {/if}
      </div>

      <div class="grid gap-2">
        <Label for="description">{$_('calendar.description')}</Label>
        <textarea
          id="description"
          bind:value={description}
          placeholder="Add description"
          class="flex min-h-[120px] w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50 resize-none"
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

<ThreeOptionDialog
  bind:open={showInviteConfirm}
  title={$_('calendar.sendInvitesTitle') || '¿Quieres enviar invitaciones por correo electrónico a los invitados de Calendario de Google?'}
  option1Label={$_('calendar.sendInvites') || 'Enviar'}
  option2Label={$_('calendar.dontSendInvites') || 'No enviar'}
  option3Label={$_('calendar.backToEdit') || 'Regresar'}
  option1Variant="default"
  option2Variant="default"
  loading={saving ? (showInviteConfirm ? 'option1' : null) : null}
  onOption1={() => doSave(true)}
  onOption2={() => doSave(false)}
  onOption3={() => { showInviteConfirm = false }}
/>

