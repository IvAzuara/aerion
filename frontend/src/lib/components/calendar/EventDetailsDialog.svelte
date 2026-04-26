<script lang="ts">
  import Icon from '@iconify/svelte'
  import * as Dialog from '$lib/components/ui/dialog'
  import { Button } from '$lib/components/ui/button'
  import { _ } from '$lib/i18n'
  import { format } from 'date-fns'
  // @ts-ignore - wailsjs models
  import { calendar } from '../../../../wailsjs/go/models'
  // @ts-ignore - wailsjs runtime
  import { BrowserOpenURL } from '../../../../wailsjs/runtime/runtime'

  interface Props {
    open?: boolean
    event: calendar.Event | null
    calendarColor?: string
    onClose?: () => void
    onDelete?: (eventId: string) => void
  }

  let {
    open = $bindable(false),
    event,
    calendarColor = '#3b82f6',
    onClose,
    onDelete,
  }: Props = $props()

  function handleJoinMeet() {
    if (event?.meetLink) {
      BrowserOpenURL(event.meetLink)
    }
  }
</script>

<Dialog.Root bind:open onOpenChange={(isOpen) => !isOpen && onClose?.()}>
  <Dialog.Content class="max-w-md p-0 overflow-hidden border-none bg-background shadow-2xl">
    {#if event}
      <!-- Colored Header -->
      <div class="h-2 w-full" style="background-color: {calendarColor}"></div>
      
      <div class="p-6 space-y-6">
        <div class="space-y-2">
          <div class="flex justify-between items-start gap-4">
            <h2 class="text-2xl font-semibold leading-tight">{event.summary || $_('calendar.noSubject')}</h2>
            <div class="flex gap-1">
              <Button variant="ghost" size="icon" class="h-8 w-8" onclick={() => onDelete?.(event.id)}>
                <Icon icon="mdi:delete-outline" class="w-5 h-5 text-muted-foreground hover:text-destructive" />
              </Button>
              <Button variant="ghost" size="icon" class="h-8 w-8" onclick={() => open = false}>
                <Icon icon="mdi:close" class="w-5 h-5 text-muted-foreground" />
              </Button>
            </div>
          </div>
          
          <div class="flex items-center gap-2 text-muted-foreground">
            <Icon icon="mdi:clock-outline" class="w-5 h-5" />
            <span class="text-sm font-medium">
              {format(new Date(event.startTime), 'EEEE, MMMM do')}
              <br/>
              {#if event.isAllDay}
                {$_('calendar.allDay')}
              {:else}
                {format(new Date(event.startTime), 'p')} – {format(new Date(event.endTime), 'p')}
              {/if}
            </span>
          </div>
        </div>

        {#if event.meetLink}
          <Button class="w-full bg-blue-600 hover:bg-blue-700 text-white gap-2 h-11" onclick={handleJoinMeet}>
            <Icon icon="logos:google-meet" class="w-5 h-5" />
            Join with Google Meet
          </Button>
        {/if}

        {#if event.location}
          <div class="flex items-start gap-3">
            <Icon icon="mdi:map-marker-outline" class="w-5 h-5 text-muted-foreground mt-0.5" />
            <span class="text-sm">{event.location}</span>
          </div>
        {/if}

        {#if event.description}
          <div class="flex items-start gap-3 pt-2 border-t border-border">
            <Icon icon="mdi:text-subject" class="w-5 h-5 text-muted-foreground mt-0.5" />
            <div class="text-sm text-muted-foreground max-h-48 overflow-y-auto whitespace-pre-wrap">
              {event.description}
            </div>
          </div>
        {/if}
        
        {#if event.attendees && event.attendees.length > 0}
          <div class="space-y-3">
            <div class="flex items-center gap-3">
              <Icon icon="mdi:account-group-outline" class="w-5 h-5 text-muted-foreground" />
              <span class="text-sm font-medium">{event.attendees.length} guests</span>
            </div>
            <div class="pl-8 space-y-2 max-h-32 overflow-y-auto">
              {#each event.attendees as guest}
                <div class="flex items-center justify-between text-sm">
                  <span class="truncate pr-2" title={guest.email}>
                    {guest.name || guest.email}
                  </span>
                  {#if guest.status === 'accepted'}
                    <Icon icon="mdi:check-circle" class="text-green-500 w-4 h-4" />
                  {:else if guest.status === 'declined'}
                    <Icon icon="mdi:close-circle" class="text-destructive w-4 h-4" />
                  {:else}
                    <Icon icon="mdi:help-circle" class="text-amber-500 w-4 h-4" />
                  {/if}
                </div>
              {/each}
            </div>
          </div>
        {/if}
      </div>
    {/if}
  </Dialog.Content>
</Dialog.Root>
