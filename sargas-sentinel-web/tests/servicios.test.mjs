import test from 'node:test'
import assert from 'node:assert/strict'
import {
  obtenerServicios, obtenerEstadoServicio, crearServicio, actualizarServicio, leerServiciosGuardados
} from '../src/api/servicios.ts'

// Pruebas aisladas: no hacen solicitudes reales ni acceden a PostgreSQL.
const base = {
  id: 12,
  nombre: 'Servicio de prueba',
  url: 'https://example.com',
  tipo: 'HTTP',
  intervalo_segundos: 15,
  timeout_segundos: 5,
  habilitado: true
}
const formulario = {
  nombre: base.nombre,
  url: base.url,
  intervalo_segundos: base.intervalo_segundos,
  timeout_segundos: base.timeout_segundos
}
function responder(objeto, status = 200) {
  return new Response(JSON.stringify(objeto), {
    status,
    headers: { 'Content-Type': 'application/json' }
  })
}
function simular(t, fetchSimulado) {
  t.mock.method(globalThis, 'fetch', fetchSimulado)
}

function almacenamiento() {
  const valores = new Map();
  return {
    get length() { return valores.size; },
    key(i) { return [...valores.keys()][i] ?? null; },
    getItem(k) { return valores.get(k) ?? null; },
    setItem(k,v) { valores.set(k,String(v)); },
    removeItem(k) { valores.delete(k); }
  };
}

test('Compartir una copia conserva su fecha original', async () => {
  globalThis.sessionStorage = almacenamiento();
  const { guardarCopia } = await import('../src/lib/cacheSesion.ts');
  const fecha = new Date(Date.now() - 60_000).toISOString();
  guardarCopia('servicios', [base], fecha);
  assert.equal(leerServiciosGuardados()?.sincronizadoEn, fecha);
});

test('Dashboard y Servicios comparten la última lista validada', async (t) => {
  globalThis.sessionStorage = almacenamiento();
  simular(t, async () => responder([base]));
  await obtenerServicios();
  assert.deepEqual(leerServiciosGuardados()?.datos, [base]);
});

test('Si la API falla se conserva la copia compartida previa', async (t) => {
  globalThis.sessionStorage = almacenamiento();
  simular(t, async () => { throw new TypeError('Failed to fetch'); });
  const previo = {
    version: 2,
    sincronizadoEn: new Date(Date.now() - 60_000).toISOString(),
    datos: [base]
  };
  sessionStorage.setItem('sargas.cache.v2.servicios', JSON.stringify(previo));
  await assert.rejects(obtenerServicios());
  assert.equal(leerServiciosGuardados()?.sincronizadoEn, previo.sincronizadoEn);
});

test('Estado validado en Dashboard queda disponible para Detalle', async (t) => {
  globalThis.sessionStorage = almacenamiento();
  const estado = {
    servicio_id: base.id, estado: 'UP', habilitado: true,
    exitos_consecutivos: 2, fallos_consecutivos: 0,
    ultima_comprobacion_en: null
  };
  simular(t, async () => responder(estado));
  await obtenerEstadoServicio(base.id);
  const { leerCopia } = await import('../src/lib/cacheSesion.ts');
  const copia = leerCopia(`detalle.${base.id}.estado`, v => v?.servicio_id === base.id);
  assert.deepEqual(copia?.datos, estado);
});

test('GET servicios: lista válida', async (t) => {
  simular(t, async (ruta) => {
    assert.equal(ruta, '/api/servicios')
    return responder([base])
  })
  assert.deepEqual(await obtenerServicios(), [base])
})

test('GET servicios: lista vacía válida', async (t) => {
  simular(t, async () => responder([]))
  assert.deepEqual(await obtenerServicios(), [])
})

test('GET servicios: rechaza objeto en lugar de lista', async (t) => {
  simular(t, async () => responder(base))
  await assert.rejects(obtenerServicios(), /lista de servicios/)
})

test('GET servicios: rechaza elemento inválido', async (t) => {
  simular(t, async () => responder([{ ...base, habilitado: 'sí' }]))
  await assert.rejects(obtenerServicios(), /servicios con datos inválidos/)
})

test('GET estado: estado válido del servicio correcto', async (t) => {
  const estado = {
    servicio_id: 12,
    estado: 'UP',
    habilitado: true,
    exitos_consecutivos: 2,
    fallos_consecutivos: 0,
    ultima_comprobacion_en: null
  }
  simular(t, async (ruta) => {
    assert.equal(ruta, '/api/servicios/12/estado')
    return responder(estado)
  })
  assert.deepEqual(await obtenerEstadoServicio(12), estado)
})

test('GET estado: rechaza estado asociado a otro ID', async (t) => {
  simular(t, async () => responder({
    servicio_id: 13,
    estado: 'UP',
    habilitado: true,
    exitos_consecutivos: 2,
    fallos_consecutivos: 0,
    ultima_comprobacion_en: null
  }))
  await assert.rejects(obtenerEstadoServicio(12), /estado inválido/)
})

test('Crear válido: interpreta y devuelve respuesta de Go', async (t) => {
  simular(t, async (ruta, opciones) => {
    assert.equal(ruta, '/api/servicios')
    assert.equal(opciones.method, 'POST')
    assert.deepEqual(JSON.parse(opciones.body), formulario)
    return responder(base, 201)
  })
  assert.deepEqual(await crearServicio(formulario), base)
})

test('Crear con ID incorrecto: rechaza respuesta', async (t) => {
  simular(t, async () => responder({ ...base, id: -1 }, 201))
  await assert.rejects(crearServicio(formulario), /creado con datos inválidos/)
})

test('Crear sin habilitado: rechaza respuesta', async (t) => {
  const { habilitado, ...incompleto } = base
  simular(t, async () => responder(incompleto, 201))
  await assert.rejects(crearServicio(formulario), /creado con datos inválidos/)
})

test('Crear HTTP 400: preserva mensaje del backend', async (t) => {
  simular(t, async () => new Response('URL inválida', { status: 400 }))
  await assert.rejects(crearServicio(formulario), /URL inválida/)
})

test('Crear JSON malformado: genera SyntaxError', async (t) => {
  simular(t, async () => new Response('{no-json', { status: 201 }))
  await assert.rejects(crearServicio(formulario), SyntaxError)
})

test('Editar válido: interpreta y devuelve respuesta de Go', async (t) => {
  simular(t, async (ruta, opciones) => {
    assert.equal(ruta, '/api/servicios/12')
    assert.equal(opciones.method, 'PUT')
    return responder({ ...base, nombre: 'Nombre editado' })
  })
  const editado = await actualizarServicio(12, { ...formulario, nombre: 'Nombre editado' })
  assert.equal(editado.id, 12)
  assert.equal(editado.nombre, 'Nombre editado')
})

test('Editar con ID incorrecto: rechaza respuesta', async (t) => {
  simular(t, async () => responder({ ...base, id: 13 }))
  await assert.rejects(actualizarServicio(12, formulario), /actualizado con datos inválidos/)
})

test('Editar con URL vacía: rechaza respuesta', async (t) => {
  simular(t, async () => responder({ ...base, url: '' }))
  await assert.rejects(actualizarServicio(12, formulario), /actualizado con datos inválidos/)
})

test('Editar HTTP 404: informa inexistencia', async (t) => {
  simular(t, async () => new Response('No existe', { status: 404 }))
  await assert.rejects(actualizarServicio(12, formulario), /ya no existe/)
})

test('Editar HTTP 503: informa error de sincronización', async (t) => {
  simular(t, async () => new Response('Error temporal', { status: 503 }))
  await assert.rejects(actualizarServicio(12, formulario), /HTTP 503/)
})
