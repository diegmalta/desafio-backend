// Carga em POST /webhook com HMAC-SHA256 do corpo bruto (igual ao servidor).
// Variáveis de ambiente: WEBHOOK_SECRET (obrigatório), BASE_URL (opcional, default http://localhost:8080), K6_CPF (opcional, 11 dígitos).
import http from 'k6/http';
import { check } from 'k6';
import { hmac } from 'k6/crypto';
import exec from 'k6/execution';

export function setup() {
  if (!__ENV.WEBHOOK_SECRET || String(__ENV.WEBHOOK_SECRET).length === 0) {
    throw new Error('WEBHOOK_SECRET tem de estar definido no ambiente (mesmo valor que no .env do servidor)');
  }
}

export const options = {
  stages: [
    { duration: '10s', target: 5 },
    { duration: '20s', target: 5 },
    { duration: '10s', target: 0 },
  ],
  thresholds: {
    http_req_failed: ['rate<0.02'],
  },
};

function buildBody() {
  const cpf = (__ENV.K6_CPF || '12345678901').replace(/\D/g, '');
  if (cpf.length !== 11) {
    throw new Error('K6_CPF tem de ter 11 dígitos');
  }
  const ts = new Date().toISOString();
  const vu = exec.vu.idInTest;
  const it = exec.vu.iterationInScenario;
  const chamado = `CH-k6-${vu}-${it}-${Date.now()}`;
  const o = {
    chamado_id: chamado,
    tipo: 'status_change',
    cpf: cpf,
    status_anterior: 'em_analise',
    status_novo: 'em_execucao',
    titulo: 'k6 — carga webhook',
    descricao: 'Evento sintético para teste de carga',
    timestamp: ts,
  };
  return JSON.stringify(o);
}

export default function () {
  const base = (__ENV.BASE_URL || 'http://localhost:8080').replace(/\/$/, '');
  const body = buildBody();
  const macHex = hmac('sha256', __ENV.WEBHOOK_SECRET, body, 'hex');
  const res = http.post(`${base}/webhook`, body, {
    headers: {
      'Content-Type': 'application/json; charset=utf-8',
      'X-Signature-256': `sha256=${macHex}`,
    },
    tags: { name: 'webhook' },
  });
  check(res, {
    'status 2xx': (r) => r.status >= 200 && r.status < 300,
  });
}
