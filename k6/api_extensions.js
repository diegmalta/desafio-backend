// Exercita novos endpoints: citizens/me, read-all, chamados summary, mapas/status, devices.
// Env: K6_JWT (obrigatório), WEBHOOK_SECRET (opcional; se definido, cria CH-k6-ext no setup), BASE_URL, K6_CPF.
import http from 'k6/http';
import { check } from 'k6';
import { hmac } from 'k6/crypto';

export function setup() {
  if (!__ENV.K6_JWT || String(__ENV.K6_JWT).trim() === '') {
    throw new Error('K6_JWT obrigatório');
  }
  const b = String(__ENV.BASE_URL || 'http://localhost:8080').replace(/\/$/, '');
  if (__ENV.WEBHOOK_SECRET) {
    const cpf = (__ENV.K6_CPF || '12345678901').replace(/\D/g, '');
    const ts = new Date().toISOString();
    const body = JSON.stringify({
      chamado_id: 'CH-k6-ext',
      tipo: 'status_change',
      cpf: cpf,
      status_anterior: 'em_analise',
      status_novo: 'em_execucao',
      titulo: 'k6 ext',
      descricao: 'seed chamado summary',
      timestamp: ts,
    });
    const macHex = hmac('sha256', __ENV.WEBHOOK_SECRET, body, 'hex');
    http.post(`${b}/webhook`, body, {
      headers: {
        'Content-Type': 'application/json; charset=utf-8',
        'X-Signature-256': `sha256=${macHex}`,
      },
    });
  }
  return { base: b };
}

export const options = {
  vus: 3,
  duration: '20s',
  thresholds: {
    http_req_failed: ['rate<0.15'],
  },
};

export default function (data) {
  const b = data.base;
  const authHdr = { Authorization: `Bearer ${String(__ENV.K6_JWT).trim()}` };

  let res = http.get(`${b}/citizens/me`, { headers: authHdr, tags: { name: 'citizens_me' } });
  check(res, { 'me 200': (r) => r.status === 200 });

  res = http.get(`${b}/mapas/status`, { headers: authHdr, tags: { name: 'mapas_status' } });
  check(res, { 'mapas 200': (r) => r.status === 200 });

  res = http.get(`${b}/notifications?limit=5&offset=0`, { headers: authHdr });
  let notifId = '';
  if (res.status === 200) {
    try {
      const j = res.json();
      if (j.items && j.items.length > 0) {
        notifId = j.items[0].id;
      }
    } catch (e) {}
  }
  if (notifId) {
    res = http.get(`${b}/notifications/${notifId}`, { headers: authHdr, tags: { name: 'notif_detail' } });
    check(res, { 'detail 200': (r) => r.status === 200 });
  }

  res = http.get(`${b}/chamados/${encodeURIComponent('CH-k6-ext')}/summary`, {
    headers: authHdr,
    tags: { name: 'chamados_summary' },
  });
  check(res, {
    'chamados 200 ou 404': (r) => r.status === 200 || r.status === 404,
  });

  res = http.post(
    `${b}/devices`,
    JSON.stringify({ token: `k6-token-${__VU}-${__ITER}`, platform: 'web' }),
    { headers: { ...authHdr, 'Content-Type': 'application/json' }, tags: { name: 'devices_register' } }
  );
  check(res, { 'devices 200': (r) => r.status === 200 });

  res = http.patch(`${b}/notifications/read-all`, null, { headers: authHdr, tags: { name: 'read_all' } });
  check(res, { 'read_all 200': (r) => r.status === 200 });
}
