import test from 'node:test'
import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import ts from 'typescript'

const fuente = await readFile(new URL('../src/api/comprobaciones.ts', import.meta.url), 'utf8')
const modulo = ts.transpileModule(fuente, {
  compilerOptions: {
    target: ts.ScriptTarget.ES2022,
    module: ts.ModuleKind.ESNext
  }
}).outputText
const { obtenerComprobaciones } = await import(`data:text/javascript;base64,${Buffer.from(modulo).toString('base64')}`)

const registro = {
  id: 7,
  servicio_id: 12,
  codigo_http: 503,
  latencia_ms: 145.5,
  error: null,
  comprobado_en: '2026-11-20T10:00:00Z'
}

function respuesta(datos, status = 200) {
  return new Response(JSON.stringify(datos), {
    status,
    headers: { 'content-type': 'application/json' }
  })
}

test('Consulta 20, 50 y 100 comprobaciones sin cambiar el endpoint', async (t) => {
  const rutas = []
  t.mock.method(globalThis, 'fetch', async (ruta) => {
    rutas.push(ruta)
    return respuesta([registro])
  })
  for (const cantidad of [20, 50, 100]) {
    assert.deepEqual(await obtenerComprobaciones(12, cantidad), [registro])
  }
  assert.deepEqual(rutas, [20, 50, 100].map((n) =>
    `/api/servicios/12/comprobaciones?limite=${n}&offset=0`))
})

test('Propaga la señal de cancelación al fetch', async (t) => {
  const control = new AbortController()
  t.mock.method(globalThis, 'fetch', async (_ruta, opciones) => {
    assert.equal(opciones.signal, control.signal)
    return respuesta([])
  })
  assert.deepEqual(await obtenerComprobaciones(12, 20, control.signal), [])
})

test('No acepta cantidades superiores a 100', async () => {
  await assert.rejects(obtenerComprobaciones(12, 101), /entre 1 y 100/)
})

test('Un error HTTP no se interpreta como lista válida', async (t) => {
  t.mock.method(globalThis, 'fetch', async () => respuesta([], 503))
  await assert.rejects(obtenerComprobaciones(12, 20), /HTTP 503/)
})

test('Rechaza registros pertenecientes a otro servicio', async (t) => {
  t.mock.method(globalThis, 'fetch', async () => respuesta([{ ...registro, servicio_id: 13 }]))
  await assert.rejects(obtenerComprobaciones(12), /datos inválidos/)
})
