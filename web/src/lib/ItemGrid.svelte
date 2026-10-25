<script lang="ts">
	import { boxName, dispositionName, forSale, type ItemListEntry } from './api.ts';
	import { formatEuro } from './money.ts';

	// By default the cards are plain links. With `onlongpress` (start page) a long press calls it instead of
	// opening the item, and while `selected` is not null a tap toggles the item via `ontoggle`.
	let {
		items,
		selected = null,
		ontoggle = () => {},
		onlongpress
	}: {
		items: ItemListEntry[];
		selected?: number[] | null;
		ontoggle?: (id: number) => void;
		onlongpress?: (id: number) => void;
	} = $props();

	// Disposition with the asking price; in the archive what became of the item and for how much.
	function status(item: ItemListEntry) {
		if (item.handedOverOn) return item.disposition === 'give_away' ? 'Verschenkt' : `Verkauft · ${formatEuro(item.saleProceeds ?? 0)}`;
		const name = dispositionName(item.disposition);
		return forSale(item.disposition) && item.askingPrice !== null ? `${name} · ${formatEuro(item.askingPrice)}` : name;
	}

	// Place and box in a muted line below the name.
	const where = (item: ItemListEntry) =>
		[item.place, item.move?.box ? boxName(item.move.box) : ''].filter(Boolean).join(' · ');

	const LONG_PRESS_MS = 500;
	// A finger moving further than this is scrolling, not pressing.
	const MOVE_TOLERANCE_PX = 10;

	let press: { x: number; y: number; timer: ReturnType<typeof setTimeout> } | null = null;
	// Set when a long press fired, so the click that follows on release neither navigates nor toggles.
	let longPressed = false;

	function pressStart(e: PointerEvent, id: number) {
		if (!onlongpress || e.button !== 0) return;
		pressEnd();
		longPressed = false;
		press = {
			x: e.clientX,
			y: e.clientY,
			timer: setTimeout(() => {
				press = null;
				longPressed = true;
				onlongpress?.(id);
			}, LONG_PRESS_MS)
		};
	}

	function pressMove(e: PointerEvent) {
		if (press && Math.hypot(e.clientX - press.x, e.clientY - press.y) > MOVE_TOLERANCE_PX) pressEnd();
	}

	function pressEnd() {
		if (press) clearTimeout(press.timer);
		press = null;
	}

	function click(e: MouseEvent, id: number) {
		if (longPressed) {
			longPressed = false;
			e.preventDefault();
			return;
		}
		if (selected) {
			e.preventDefault();
			ontoggle(id);
		}
	}
</script>

<svelte:window onscroll={pressEnd} />

<!-- Depending on the theme a list (Minimal) or tiles (Neon, Soft); app.css alone decides. -->
<ul class="item-grid" class:selecting={!!selected} class:long-press={!!onlongpress}>
	{#each items as item (item.id)}
		{@const checked = !!selected?.includes(item.id)}
		<li>
			<a
				class="card"
				class:selected={checked}
				href="/items/{item.id}"
				role={selected ? 'checkbox' : undefined}
				aria-checked={selected ? checked : undefined}
				onclick={(e) => click(e, item.id)}
				onpointerdown={(e) => pressStart(e, item.id)}
				onpointermove={pressMove}
				onpointerup={pressEnd}
				onpointercancel={pressEnd}
				onpointerleave={pressEnd}
				oncontextmenu={(e) => {
					// Android opens the link menu on a long press; the long press selects instead.
					if (longPressed || press) e.preventDefault();
				}}
			>
				{#if selected}<span class="card-check" aria-hidden="true"></span>{/if}
				<img src={item.thumbnailUrl} alt="" loading="lazy" draggable="false" />
				<span class="card-text">
					<span class="card-name">{item.name}</span>
					{#if where(item)}<span class="card-place">{where(item)}</span>{/if}
				</span>
				<span class="disposition" data-disposition={item.disposition ?? 'undecided'}>{status(item)}</span>
			</a>
		</li>
	{/each}
</ul>
