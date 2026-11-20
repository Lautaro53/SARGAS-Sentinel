<script lang="ts">
  import { crearServicio, actualizarServicio } from "../api/servicios";
  import type { Servicio } from "../types/servicio";
  import Button from "./ui/Button.svelte";

  type Props = {
    servicio?: Servicio | null;
    onCreado: (servicio: Servicio) => void;
    onActualizado: (servicio: Servicio) => void;
    onCancelar: () => void;
  };

  let {
    servicio = null,
    onCreado,
    onActualizado,
    onCancelar,
  }: Props = $props();

  let nombre = $state("");
  let direccion = $state("");
  let intervalo = $state(15);
  let timeout = $state(5);

  $effect(() => {
    nombre = servicio?.nombre ?? "";
    direccion = servicio?.url ?? "";
    intervalo = servicio?.intervalo_segundos ?? 15;
    timeout = servicio?.timeout_segundos ?? 5;
  });

  let enviando = $state(false);
  let error = $state("");

  async function guardar(evento: SubmitEvent) {
    evento.preventDefault();

    if (enviando) return;

    error = "";

    const nombreLimpio = nombre.trim();
    const urlLimpia = direccion.trim();

    // Validar el nombre.
    if (nombreLimpio.length === 0 || Array.from(nombreLimpio).length > 120) {
      error = "El nombre debe contener entre 1 y 120 caracteres.";
      return;
    }

    // Validar la URL HTTP/HTTPS.
    try {
      const url = new URL(urlLimpia);

      if (url.protocol !== "http:" && url.protocol !== "https:") {
        error = "La URL debe utilizar HTTP o HTTPS.";
        return;
      }
    } catch {
      error = "Introduce una URL válida.";
      return;
    }

    if (
      !Number.isInteger(intervalo) ||
      !Number.isInteger(timeout) ||
      intervalo <= 0 ||
      timeout <= 0
    ) {
      error = "El intervalo y el timeout deben ser enteros positivos.";
      return;
    }

    enviando = true;

    try {
      const datos = {
        nombre: nombreLimpio,
        url: urlLimpia,
        intervalo_segundos: intervalo,
        timeout_segundos: timeout,
      };

      if (servicio !== null) {
        const actualizado = await actualizarServicio(servicio.id, datos);

        onActualizado(actualizado);
      } else {
        const nuevo = await crearServicio(datos);

        onCreado(nuevo);
      }
    } catch (e) {
      error =
        e instanceof Error ? e.message : "No se pudo registrar el servicio";
    } finally {
      enviando = false;
    }
  }
</script>

<section id="formulario-servicio" class="form-panel">
  <h2>{servicio ? "Editar servicio" : "Nuevo servicio"}</h2>

  <p class="description">
    Configura un servicio para comenzar su monitoreo HTTP.
  </p>

  <form onsubmit={guardar}>
    <div class="field-row">
      <div class="field">
        <label for="nombre-servicio">Nombre</label>

        <input
          id="nombre-servicio"
          type="text"
          maxlength="120"
          placeholder="API principal"
          bind:value={nombre}
          required
          disabled={enviando}
        />
      </div>

      <div class="field">
        <label for="url-servicio">URL</label>

        <input
          id="url-servicio"
          type="url"
          placeholder="https://example.com"
          bind:value={direccion}
          required
          disabled={enviando}
        />
      </div>
    </div>

    <div class="field-row">
      <div class="field">
        <label for="intervalo">Intervalo (segundos)</label>

        <input
          id="intervalo"
          type="number"
          min="1"
          step="1"
          bind:value={intervalo}
          required
          disabled={enviando}
        />
      </div>

      <div class="field">
        <label for="timeout">Timeout (segundos)</label>

        <input
          id="timeout"
          type="number"
          min="1"
          step="1"
          bind:value={timeout}
          required
          disabled={enviando}
        />
      </div>
    </div>

    <p class="hint">Tipo: HTTP/HTTPS. El servicio comenzará habilitado.</p>

    {#if error}
      <p class="error" role="alert">{error}</p>
    {/if}

    <div class="actions">
      <Button type="button" variant="secondary" disabled={enviando} onClick={onCancelar}>
        Cancelar
      </Button>

      <Button type="submit" variant="primary" loading={enviando}>
        {enviando
          ? "Guardando..."
          : servicio
            ? "Guardar cambios"
            : "Registrar servicio"}
      </Button>
    </div>
  </form>
</section>

<style>
  .form-panel {
    margin-top: 0;
    margin-bottom: 16px;
    padding: 20px;
    border: 1px solid var(--color-border);
    border-radius: var(--radius-card);
    background: var(--color-surface);
    scroll-margin-top: 88px;
  }

  h2 {
    margin: 0 0 8px;
    font-size: 18px;
  }

  .description,
  .hint {
    color: var(--color-muted);
    font-size: 13px;
  }

  form {
    display: grid;
    gap: 14px;
    margin-top: 16px;
  }

  .field {
    display: flex;
    flex-direction: column;
    gap: 6px;
    min-width: 0;
  }

  label {
    font-size: 13px;
    font-weight: 500;
  }

  input {
    width: 100%;
    min-height: 40px;
    padding: 9px 12px;
    color: var(--color-text);
    background: var(--color-background);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-control);
    font: inherit;
    font-size: 13px;
  }

  input:focus,
  input:focus-visible {
    outline: none;
    border-color: var(--color-primary-hover);
    box-shadow: 0 0 0 1px rgba(0, 102, 255, 0.2);
  }

  .field-row {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 14px;
  }
  .hint {
    margin: 0;
  }

  .error {
    color: var(--color-down);
    font-size: 13px;
  }

  .actions {
    display: flex;
    justify-content: flex-end;
    flex-wrap: wrap;
    gap: 10px;
  }

  @media (max-width: 700px) {
    .field-row {
      grid-template-columns: 1fr;
    }

    .form-panel {
      padding: 16px;
      scroll-margin-top: 76px;
    }
  }

  @media (max-width: 520px) {
    .field-row {
      grid-template-columns: 1fr;
    }
  }
</style>
