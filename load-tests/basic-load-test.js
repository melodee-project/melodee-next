#!/usr/bin/env node

import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate } from 'k6/metrics';

// Define custom metrics
const successRate = new Rate('success_rate');

// Options for the test
export let options = {
  stages: [
    { duration: '2m', target: 10 },    // Ramp up to 10 users
    { duration: '5m', target: 10 },    // Stay at 10 users
    { duration: '2m', target: 20 },    // Ramp up to 20 users
    { duration: '5m', target: 20 },    // Stay at 20 users
    { duration: '2m', target: 50 },    // Ramp up to 50 users
    { duration: '10m', target: 50 },   // Stay at 50 users
    { duration: '2m', target: 0 },     // Ramp down to 0 users
  ],
  thresholds: {
    // Success rate should be above 99%
    'success_rate': ['rate>0.99'],
    // 95% of requests should be faster than 500ms
    'http_req_duration': ['p(95)<500'],
    // 99% of requests should be faster than 1000ms
    'http_req_duration': ['p(99)<1000'],
  },
};

// User credentials for authentication testing
const USERS = [
  { username: 'user1', password: 'password1' },
  { username: 'user2', password: 'password2' },
  { username: 'user3', password: 'password3' },
];

export default function () {
  // Randomly select a user for this iteration
  const user = USERS[Math.floor(Math.random() * USERS.length)];
  
  // Base URL for the API
  const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080/rest';
  
  // Authenticate user
  const authParams = {
    username: user.username,
    password: user.password,
    c: 'melodee-k6-test',  // Client ID
    v: '1.16.1',          // OpenSubsonic version
    f: 'json',            // Format
  };
  
  // Construct authentication parameters with encoded password
  const authUrl = `${BASE_URL}/getLicense.view?u=${authParams.username}&p=enc:${btoa(authParams.password)}&v=${authParams.v}&c=${authParams.c}&f=${authParams.f}`;
  
  const authResponse = http.get(authUrl);
  successRate.add(authResponse.status === 200);
  
  if (authResponse.status !== 200) {
    console.log(`Auth failed for ${user.username}: ${authResponse.status}`);
    return; // Skip rest of test if auth fails
  }
  
  // Test common browsing endpoints
  const browsingTests = [
    `${BASE_URL}/getArtists.view?u=${user.username}&p=enc:${btoa(authParams.password)}&v=${authParams.v}&c=${authParams.c}&f=${authParams.f}`,
    `${BASE_URL}/getIndexes.view?u=${user.username}&p=enc:${btoa(authParams.password)}&v=${authParams.v}&c=${authParams.c}&f=${authParams.f}`,
    `${BASE_URL}/getGenres.view?u=${user.username}&p=enc:${btoa(authParams.password)}&v=${authParams.v}&c=${authParams.c}&f=${authParams.f}`,
  ];
  
  // Make random browsing requests
  const randomBrowsingUrl = browsingTests[Math.floor(Math.random() * browsingTests.length)];
  const browsingResponse = http.get(randomBrowsingUrl);
  successRate.add(browsingResponse.status === 200);
  
  // Simulate search activity
  const searchTerms = ['rock', 'pop', 'jazz', 'classical', 'electronic'];
  const searchTerm = searchTerms[Math.floor(Math.random() * searchTerms.length)];
  const searchUrl = `${BASE_URL}/search3.view?u=${user.username}&p=enc:${btoa(authParams.password)}&query=${encodeURIComponent(searchTerm)}&v=${authParams.v}&c=${authParams.c}&f=${authParams.f}`;
  
  const searchResponse = http.get(searchUrl);
  successRate.add(searchResponse.status === 200);
  
  // Simulate streaming activity (we'll use a mock ID since we don't have actual media)
  const streamingUrl = `${BASE_URL}/stream.view?u=${user.username}&p=enc:${btoa(authParams.password)}&id=1&v=${authParams.v}&c=${authParams.c}&f=${authParams.f}`;
  
  // Don't check success for streaming as it may return 404 for non-existent files
  http.get(streamingUrl);
  
  // Add a random delay between requests to simulate realistic usage
  sleep(Math.random() * 2 + 1); // Sleep between 1-3 seconds
}

// Setup function to initialize before test runs
export function setup() {
  console.log('Starting Melodee performance test...');
  return { testStartTime: new Date() };
}

// Teardown function to run after test completes
export function teardown(data) {
  console.log(`Test completed at: ${new Date()}`);
  console.log(`Test ran for: ${options.stages.reduce((acc, stage) => acc + getDurationInSeconds(stage.duration), 0)} seconds`);
}

// Helper function to convert duration string to seconds
function getDurationInSeconds(durationString) {
  const matches = durationString.match(/(\d+)([mhs])/);
  if (!matches) return 0;
  
  const value = parseInt(matches[1]);
  const unit = matches[2];
  
  switch(unit) {
    case 'h': return value * 3600;
    case 'm': return value * 60;
    case 's': return value;
    default: return 0;
  }
}