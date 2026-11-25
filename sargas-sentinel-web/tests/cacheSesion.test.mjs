import test from 'node:test';
import assert from 'node:assert/strict';
import { leerCopia, guardarCopia, invalidarCopias, esPagina } from '../src/lib/cacheSesion.ts';

function almacenamiento() {
  const datos = new Map();
  return {
    get length() { return datos.size; },
    key(i) { return [...datos.keys()][i] ?? null; },
    getItem(k) { return datos.get(k) ?? null; },
    setItem(k,v) { datos.set(k,String(v)); },
    removeItem(k) { datos.delete(k); },
  };
}
const validaLista = v => Array.isArray(v) && v.every(x => Number.isSafeInteger(x));

test('recupera solamente copias validadas con fecha', () => {
  globalThis.sessionStorage = almacenamiento();
  const fecha = guardarCopia('prueba', [1,2]);
  assert.deepEqual(leerCopia('prueba', validaLista), { datos: [1,2], sincronizadoEn: fecha });
  assert.equal(leerCopia('prueba', () => false), null);
});

test('rechaza caché obsoleta o manipulada', () => {
  globalThis.sessionStorage = almacenamiento();
  sessionStorage.setItem('sargas.cache.v2.antigua', JSON.stringify({ version: 2, sincronizadoEn: '2020-01-01T00:00:00.000Z', datos: [1] }));
  sessionStorage.setItem('sargas.cache.v2.version', JSON.stringify({ version: 1, sincronizadoEn: new Date().toISOString(), datos: [1] }));
  assert.equal(leerCopia('antigua', validaLista), null);
  assert.equal(leerCopia('version', validaLista), null);
});

test('verifica paginación y pertenencia al servicio', () => {
  const validador = v => typeof v === 'object' && v !== null && Number.isSafeInteger(v.servicio_id);
  assert.equal(esPagina({total: 4, registros: [{servicio_id: 7}]}, validador, 7), true);
  assert.equal(esPagina({total: 4, registros: [{servicio_id: 8}]}, validador, 7), false);
  assert.equal(esPagina({total: -1, registros: []}, validador), false);
});

test('invalida caché compartida y de dashboard', () => {
  globalThis.sessionStorage = almacenamiento();
  guardarCopia('servicios', [1]);
  sessionStorage.setItem('sargas.dashboard.v1', 'valor');
  sessionStorage.setItem('otro', 'conservar');
  invalidarCopias();
  assert.equal(leerCopia('servicios', validaLista), null);
  assert.equal(sessionStorage.getItem('sargas.dashboard.v1'), null);
  assert.equal(sessionStorage.getItem('otro'), 'conservar');
});
