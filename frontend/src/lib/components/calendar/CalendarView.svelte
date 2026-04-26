<script lang="ts">
  import Icon from '@iconify/svelte'
  import { onMount } from 'svelte'
  import { _ } from '$lib/i18n'
  import { Button } from '$lib/components/ui/button'
  import * as AlertDialog from '$lib/components/ui/alert-dialog'
  import { navigationStore } from '$lib/stores/navigation.svelte'
  import { accountStore } from '$lib/stores/accounts.svelte'
  import EventDetailsDialog from './EventDetailsDialog.svelte'
  // @ts-ignore - wailsjs
  import { GetCalendars, GetCalendarEvents, SyncCalendars, DeleteCalendarEvent } from '../../../../wailsjs/go/app/App'
  // @ts-ignore - wailsjs models
  import { calendar } from '../../../../wailsjs/go/models'
  import { format, startOfMonth, endOfMonth, startOfWeek, endOfWeek, eachDayOfInterval, isSameMonth, isSameDay, addMonths, subMonths } from 'date-fns'

  let calendars = $state<calendar.Calendar[]>([])
  let events = $state<calendar.Event[]>([])
  let currentMonth = $state(new Date())
  let loading = $state(false)

  // Dialog state
  let showDetails = $state(false)
  let selectedEvent = $state<calendar.Event | null>(null)
  let showDeleteConfirm = $state(false)

  function openEvent(event: calendar.Event) {
    selectedEvent = event
    showDetails = true
  }

  function requestDeleteEvent() {
    showDeleteConfirm = true
  }

  async function confirmDeleteEvent() {
    if (!selectedEvent) return
    try {
      await DeleteCalendarEvent(selectedEvent.calendarId, selectedEvent.id)
      showDetails = false
      showDeleteConfirm = false
      await loadData()
    } catch (err) {
      console.error('Failed to delete event:', err)
    }
  }

  async function loadData() {
    loading = true
    try {
      calendars = await GetCalendars()
      
      const start = startOfWeek(startOfMonth(currentMonth))
      const end = endOfWeek(endOfMonth(currentMonth))
      
      let allEvents: calendar.Event[] = []
      for (const cal of calendars) {
        if (cal.enabled) {
          const calEvents = await GetCalendarEvents(cal.id, start.toISOString(), end.toISOString())
          allEvents = [...allEvents, ...(calEvents || [])]
        }
      }
      events = allEvents
    } catch (err) {
      console.error('Failed to load calendar data:', err)
    } finally {
      loading = false
    }
  }

  onMount(() => {
    loadData()
  })

  function nextMonth() {
    currentMonth = addMonths(currentMonth, 1)
    loadData()
  }

  function prevMonth() {
    currentMonth = subMonths(currentMonth, 1)
    loadData()
  }

  function goToToday() {
    currentMonth = new Date()
    loadData()
  }

  async function handleSync() {
    loading = true
    try {
      await SyncCalendars()
      // Wait a bit for background sync to do some work
      setTimeout(async () => {
        await loadData()
        loading = false
      }, 3000)
    } catch (err) {
      console.error('Manual sync failed:', err)
      loading = false
    }
  }

  const days = $derived(eachDayOfInterval({
    start: startOfWeek(startOfMonth(currentMonth)),
    end: endOfWeek(endOfMonth(currentMonth))
  }))

  function getEventsForDay(day: Date) {
    return events.filter(e => isSameDay(new Date(e.startTime), day))
  }
</script>

<div class="flex flex-col h-full bg-background">
  <!-- Calendar Header -->
  <header class="flex items-center justify-between px-6 py-4 border-b border-border">
    <div class="flex items-center gap-4">
      <h1 class="text-xl font-semibold">{format(currentMonth, 'MMMM yyyy')}</h1>
      <div class="flex items-center bg-muted rounded-md p-1">
        <button class="p-1 hover:bg-background rounded transition-colors" onclick={prevMonth}>
          <Icon icon="mdi:chevron-left" class="w-5 h-5" />
        </button>
        <button class="px-3 py-1 text-sm font-medium hover:bg-background rounded transition-colors" onclick={goToToday}>
          {$_('calendar.today')}
        </button>
        <button class="p-1 hover:bg-background rounded transition-colors" onclick={nextMonth}>
          <Icon icon="mdi:chevron-right" class="w-5 h-5" />
        </button>
      </div>
    </div>
    
    <div class="flex items-center gap-2">
      <Button variant="outline" size="sm" onclick={handleSync} disabled={loading}>
        <Icon icon="mdi:sync" class="w-4 h-4 mr-2 {loading ? 'animate-spin' : ''}" />
        {$_('common.sync')}
      </Button>
      <Button size="sm" onclick={() => navigationStore.setView('mail')}>
        <Icon icon="mdi:email" class="w-4 h-4 mr-2" />
        {$_('common.back')}
      </Button>
    </div>
  </header>

  <!-- Calendar Grid -->
  <div class="flex-1 overflow-hidden flex flex-col relative">
    {#if !loading && calendars.length === 0}
      <div class="absolute inset-0 z-10 flex flex-col items-center justify-center bg-background/80 backdrop-blur-sm p-4 text-center">
        <Icon icon="mdi:calendar-alert" class="w-16 h-16 text-muted-foreground mb-4" />
        <h2 class="text-xl font-semibold mb-2">{$_('calendar.noCalendarsFound')}</h2>
        <p class="text-muted-foreground max-w-md mb-6">
          To see your Google Calendar events, you must connect your account using <strong>OAuth2</strong>. App Passwords are not supported for the Calendar API.
        </p>
        <Button onclick={() => navigationStore.setView('mail')}>
          Go to Accounts
        </Button>
      </div>
    {/if}
    <!-- Day Names -->
    <div class="grid grid-cols-7 border-b border-border bg-muted/30">
      {#each ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'] as day}
        <div class="py-2 text-center text-xs font-semibold text-muted-foreground uppercase tracking-wider">
          {day}
        </div>
      {/each}
    </div>

    <!-- Days Grid -->
    <div class="flex-1 grid grid-cols-7 grid-rows-6 overflow-auto">
      {#each days as day}
        <div 
          class="min-h-[100px] border-r border-b border-border p-2 cursor-pointer hover:bg-muted/5 transition-colors {isSameMonth(day, currentMonth) ? '' : 'bg-muted/10'}"
        >
          <div class="flex justify-between items-start mb-1">
            <span class="text-sm font-medium {isSameDay(day, new Date()) ? 'bg-primary text-primary-foreground w-6 h-6 flex items-center justify-center rounded-full' : 'text-muted-foreground'}">
              {format(day, 'd')}
            </span>
          </div>
          
          <div class="space-y-1">
            {#each getEventsForDay(day) as event}
              <button 
                class="w-full text-left px-2 py-0.5 text-[10px] sm:text-xs rounded border truncate hover:brightness-95 transition-all flex items-center gap-1"
                style="background-color: {calendars.find(c => c.id === event.calendarId)?.color || '#3b82f6'}22; border-color: {calendars.find(c => c.id === event.calendarId)?.color || '#3b82f6'}"
                onclick={(e) => { e.stopPropagation(); openEvent(event) }}
                title={event.summary}
              >
                {#if !event.isAllDay}
                  <span class="font-bold opacity-70 whitespace-nowrap">{format(new Date(event.startTime), 'HH:mm')}</span>
                {/if}
                <span class="truncate">{event.summary || $_('calendar.noSubject')}</span>
                {#if event.meetLink}
                  <Icon icon="mdi:video" class="w-3 h-3 ml-auto opacity-70" />
                {/if}
              </button>
            {/each}
          </div>
        </div>
      {/each}
    </div>
  </div>
</div>

<EventDetailsDialog
  bind:open={showDetails}
  event={selectedEvent}
  calendarColor={calendars.find(c => c.id === selectedEvent?.calendarId)?.color}
  onDelete={requestDeleteEvent}
/>

<AlertDialog.Root bind:open={showDeleteConfirm}>
  <AlertDialog.Content>
    <AlertDialog.Header>
      <AlertDialog.Title>{$_('calendar.deleteEvent')}</AlertDialog.Title>
      <AlertDialog.Description>
        {$_('dialog.deleteDescription')}
      </AlertDialog.Description>
    </AlertDialog.Header>
    <AlertDialog.Footer>
      <AlertDialog.Cancel>{$_('common.cancel')}</AlertDialog.Cancel>
      <AlertDialog.Action onclick={confirmDeleteEvent} class="bg-destructive text-destructive-foreground hover:bg-destructive/90">
        {$_('common.delete')}
      </AlertDialog.Action>
    </AlertDialog.Footer>
  </AlertDialog.Content>
</AlertDialog.Root>
