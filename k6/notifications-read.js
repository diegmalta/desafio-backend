// Carga em GET /notifications com JWT Bearer (HS256, preferred_username = CPF).
// O cidadão pode ainda não existir na base: a API responde 200 com lista vazia; com cidadão criado por webhook, exercita listagem real.
// Variáveis: K6_JWT (obrigatório), BASE_URL (opcional).
import http from 'k6/http';
import { check } from 'k6';
import exec from 'k6/execution';

export function setup() {
  if (!__ENV.K6_JWT || String(__ENV.K6_JWT).trim() === '') {
    throw new Error(
      'K6_JWT tem de estar definido (token HS256 com preferred_username = CPF de 11 dígitos, exp no futuro; iss/aud se usares JWT_ISS/JWT_AUD no servidor)'
    );
  }
}

export const options = {
  stages: [
    { duration: '10s', target: 10 },
    { duration: '20s', target: 10 },
    { duration: '10s', target: 0 },
  ],
  thresholds: {
    http_req_failed: ['rate<0.02'],
  },
};

export default function () {
  const base = (__ENV.BASE_URL || 'http://localhost:8080').replace(/\/$/, '');
  const token = String(__ENV.K6_JWT).trim();
  const limit = 20;
  const offset = (exec.vu.iterationInScenario * limit) % 200;
  const res = http.get(`${base}/notifications?limit=${limit}&offset=${offset}`, {
    headers: {
      Authorization: `Bearer ${token}`,
    },
    tags: { name: 'notifications_list' },
  });
  check(res, {
    'status 200': (r) => r.status === 200,
    'json items': (r) => {
      if (r.status !== 200) return false;
      try {
        const j = r.json();
        return j && typeof j.items !== 'undefined' && typeof j.total !== 'undefined';
      } catch (e) {
        return false;
      }
    },
  });
}
