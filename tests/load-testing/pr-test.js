import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  scenarios: {
    create_pr: {
      executor: 'constant-arrival-rate',
      exec: 'create_pr',
      rate: 5,
      timeUnit: '1s',
      duration: '1m',
      preAllocatedVUs: 10,
      maxVUs: 20,
    },
    get_team: {
      executor: 'constant-arrival-rate',
      exec: 'get_team',
      rate: 3,
      timeUnit: '1s',
      duration: '1m',
      preAllocatedVUs: 5,
      maxVUs: 10,
    },
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

// --- Создание Pull Request ---
export function create_pr() {
  const url = `${BASE_URL}/pullRequest/create`;

  const payload = JSON.stringify({
    pull_request_id: `pr-${__VU}-${Date.now()}`,
    pull_request_name: 'load-test-pr',
    author_id: 'u1',
  });

  const params = {
    headers: {
      'Content-Type': 'application/json',
    },
  };

  const res = http.post(url, payload, params);

  check(res, {
    'status is 201 or 404/409': (r) => [201, 404, 409].includes(r.status),
  });

  sleep(0.1);
}

// --- Получение команды по имени ---
export function get_team() {
  const teamName = 'payments'; // можно заменить на любую существующую команду
  const url = `${BASE_URL}/team/get?team_name=${teamName}`;

  const res = http.get(url);

  check(res, {
    'status is 200 or 404': (r) => [200, 404].includes(r.status),
    'response has members array': (r) => {
      if (r.status !== 200) return true;
      try {
        const data = r.json();
        return Array.isArray(data.members);
      } catch (e) {
        return false;
      }
    },
  });

  sleep(0.1);
}
