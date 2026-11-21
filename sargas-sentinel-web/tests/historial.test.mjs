import test from 'node:test'
import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import ts from 'typescript'

function transpilar(fuente) {
  return ts.transpileModule(fuente, {
    compilerOptions: { target: ts.ScriptTarget.ES2022, module: ts.ModuleKind.ESNext }
  }).outputText
}
function modulo(texto) { return `data:text/javascript;base64,${Buffer.from(texto).toString('base64')}` }

const comprobaciones = modulo(transpilar(await readFile(new URL('../src/api/comprobaciones.ts', import.meta.url), 'utf8')))
const incidentes = modulo(transpilar(await readFile(new URL('../src/api/incidentes.ts', import.meta.url), 'utf8')))
const fuenteHistorial = transpilar(await readFile(new URL('../src/api/historial.ts', import.meta.url), 'utf8'))
  .replace("from './comprobaciones'", `from '${comprobaciones}'`)
  .replace("from './incidentes'", `from '${incidentes}'`)
const { consultarHistorialComprobaciones, consultarHistorialIncidentes } = await import(modulo(fuenteHistorial))

const check = {
  id: 5, servicio_id: 3, codigo_http: 303, latencia_ms: 125.5,
  error: null, comprobado_en: '2026-11-20T10:00:00Z'
}
const incident = {
  id: 9, servicio_id: 3, iniciado_en: '2026-11-20T10:00:00Z', cerrado_en: null,
  evidencia_inicio: 'HTTP 303', evidencia_cierre: null,
  nombre_servicio: 'Prueba', servicio_archivado: false
}
const respuesta = (datos, status = 200) => new Response(JSON.stringify(datos), {
  status, headers: { 'content-type': 'application/json' }
})

test('Comprobaciones: consulta paginada con filtros y totales', async (t) => {
  const control = new AbortController()
  t.mock.method(globalThis, 'fetch', async (ruta, opciones) => {
    assert.equal(opciones.signal, control.signal)
    const url = new URL(ruta, 'http://localhost')
    assert.equal(url.pathname, '/api/servicios/3/comprobaciones/consulta')
    assert.equal(url.searchParams.get('limite'), '10')
    assert.equal(url.searchParams.get('offset'), '20')
    assert.equal(url.searchParams.get('codigo'), '3xx')
    assert.equal(url.searchParams.get('orden'), 'latencia_desc')
    assert.equal(url.searchParams.get('latencia_min'), '100')
    assert.equal(url.searchParams.get('latencia_max'), '200')
    return respuesta({ registros: [check], total: 21 })
  })
  const datos = await consultarHistorialComprobaciones(3, 3, 10,
    { codigo: '3xx', orden: 'latencia_desc', latenciaMin: 100, latenciaMax: 200 }, control.signal)
  assert.deepEqual(datos, { registros: [check], total: 21 })
})

test('Incidentes: global y de servicio tienen rutas independientes', async (t) => {
  const rutas = []
  t.mock.method(globalThis, 'fetch', async (ruta) => {
    rutas.push(ruta)
    return respuesta({ registros: [incident], total: 17 })
  })
  const filtros = { estado: 'abiertos', orden: 'fecha_asc' }
  assert.equal((await consultarHistorialIncidentes(3, 2, 10, filtros)).total, 17)
  assert.equal((await consultarHistorialIncidentes(null, 1, 10, filtros)).total, 17)
  assert.match(rutas[0], /^\/api\/servicios\/3\/incidentes\/consulta\?limite=10&offset=10/)
  assert.match(rutas[1], /^\/api\/incidentes\/consulta\?limite=10&offset=0/)
})

test('Rechaza totales inventados e identificadores de otro servicio', async (t) => {
  t.mock.method(globalThis, 'fetch', async () => respuesta({ registros: [check], total: -3 }))
  await assert.rejects(consultarHistorialComprobaciones(3, 1, 10,
    { codigo: 'todos', orden: 'fecha_desc', latenciaMin: null, latenciaMax: null }), /inválida/)
})

test('Rechaza latencia mínima superior a la máxima sin enviar petición', async () => {
  await assert.rejects(consultarHistorialComprobaciones(3, 1, 10,
    { codigo: 'todos', orden: 'fecha_desc', latenciaMin: 700, latenciaMax: 200 }), /mínima/)
})

test('Propaga errores HTTP sin convertirlos en registros vacíos', async (t) => {
  t.mock.method(globalThis, 'fetch', async () => respuesta({}, 503))
  await assert.rejects(consultarHistorialIncidentes(null, 1, 10,
    { estado: 'todos', orden: 'fecha_desc' }), /503/)
})

test('Incidentes: rango por fechas en el calendario local', async (t) => {
  t.mock.method(globalThis, 'fetch', async (ruta) => {
    const url = new URL(ruta, 'http://localhost')
    assert.equal(Date.parse(url.searchParams.get('desde')), new Date(2026, 10, 1).getTime())
    assert.equal(Date.parse(url.searchParams.get('hasta')), new Date(2026, 10, 21).getTime())
    return respuesta({ registros: [], total: 0 })
  })
  await consultarHistorialIncidentes(null, 1, 10,
    { estado: 'todos', orden: 'fecha_desc', desde: '2026-11-01', hasta: '2026-11-20' })
})
