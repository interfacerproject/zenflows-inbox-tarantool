#!/usr/bin/env node

// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2022-2023 Dyne.org foundation <foundation@dyne.org>.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

import sign from "./sign_graphql.mjs"
import { zencode_exec } from 'zenroom';
import axios from 'axios';

const PIPPO_EDDSA = "EtJtSqAG9mVHfKrKduS6aeyAE6okGXrfMW8fEQ6eqenh"
const PIPPO_ID = "062TE0H7591KJCVT3DDEMDBF0R"
const PLUTO_EDDSA = "2n4TEhoQ8ZwedJoUuJNbxv5W1cr5wHFYPcQmkk1EWj4t"
const PLUTO_ID = "062TE0YPJD392CS1DPV9XWMDXC"
const PAPERINO_EDDSA = "H7sbugVBZbmRX6M75WpzCi5vVVtaxvfLhovDijRAnZj"
const PAPERINO_ID = "062TE18QJSQJ1PY6G1M7783148"

const url0="http://inbox0"
const url1="http://inbox1"
//const url="https://gateway0.interfacer.dyne.org/inbox"

const signRequest = async (json, key) => {
  const data = `{"gql": "${Buffer.from(json, 'utf8').toString('base64')}"}`
  const keys = `{"keyring": {"eddsa": "${key}"}}`
  const {result} = await zencode_exec(sign(), {data, keys});
  return {
    'zenflows-sign': JSON.parse(result).eddsa_signature
  }
}

const useLikeObject = async () => {
  const request = {
    "@context": "https://www.w3.org/ns/activitystreams",
    "type": "Like",
    "actor": `${url0}/person/062TE0H7591KJCVT3DDEMDBF0R`,
    "object": `${url1}/economicresource/062SE9RG34DDHTEHHRWY8VKCJW`,
    "published": "2014-09-30T12:34:56Z"
  }
  const requestJSON = JSON.stringify(request)
  const requestHeaders = await signRequest(requestJSON, PIPPO_EDDSA);
  const config = {
    headers: requestHeaders
  };

  const result = await axios.post(`${url0}/person/062TE0H7591KJCVT3DDEMDBF0R/outbox`, request, config);
  return result
}

const useFollowActivity = async () => {
  const request = {
    "@context": "https://www.w3.org/ns/activitystreams",
    "type": "Follow",
    "actor": `${url0}/person/${PIPPO_ID}`,
    "object": `${url0}/person/${PAPERINO_ID}`,
    "published": "2014-09-30T12:34:56Z"
  }
  const requestJSON = JSON.stringify(request)
  const requestHeaders = await signRequest(requestJSON, PIPPO_EDDSA);
  const config = {
    headers: requestHeaders
  };

  const result = await axios.post(`${url0}/person/${PIPPO_ID}/outbox`, request, config);
  return result
}

//console.log(await useLikeObject())
console.log(await useFollowActivity())
