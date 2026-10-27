<script lang="ts">
    import { ShieldCheck } from "@lucide/svelte";
</script>

<footer class="app-footer">
    <span>© {new Date().getFullYear()} SARGAS Sentinel</span>

    <div class="footer-description">
        <ShieldCheck size={15} strokeWidth={1.8} aria-hidden="true" />
        <span>Monitoreo de servicios HTTP</span>
    </div>
</footer>

<style>
    .app-footer {
        display: flex;
        align-items: center;
        justify-content: space-between;
        flex-wrap: wrap;
        gap: 12px;

        min-height: 65px;
        padding: 0;

        border-top: 1px solid var(--color-border);

        color: var(--color-muted);
        font-size: 12px;
    }

    .footer-description {
        display: flex;
        align-items: center;
        gap: 8px;
    }

    .footer-description :global(svg) {
        color: var(--color-primary);
    }

    @media (max-width: 600px) {
        .app-footer {
            flex-direction: column;
            align-items: flex-start;
        }
    }
</style>
