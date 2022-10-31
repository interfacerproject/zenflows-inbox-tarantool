#!/usr/bin/env node
import sign from "./sign_graphql.mjs"
import { zencode_exec } from 'zenroom';
import axios from 'axios';

const PIPPO_EDDSA = "AEkxhh4aFV1eG88FY8LjZSMyJXgmynUdWzUPV6tCHwqn"
const PIPPO_ID = "0620YFM3DCC74PEK6VYH32EF10"
const PLUTO_EDDSA = "6M3sJz56689qf67xhKyfT3Lf19eiq1xWXhmtzBipdW3n"
const PLUTO_ID = "061TQRJG3RP9YN9HVXH6BMXK5W"
const PAPERINO_EDDSA = "88XLEXAkTdxdm4r8V5gYFPQxvqgMvWu4EHXKSMbXenzC"
const PAPERINO_ID = "0620WKRGKNHDVRHGFYWPYM0GQ0"

const url="http://localhost:8080"

const signRequest = async (json, key) => {
	const data = `{"gql": "${Buffer.from(json, 'utf8').toString('base64')}"}`
    const keys = `{"keyring": {"eddsa": "${key}"}}`
	const {result} = await zencode_exec(sign(), {data, keys});
	return {
		'zenflows-sign': JSON.parse(result).eddsa_signature
	}
}

const sendMessage = async (message) => {
    const request = {
        sender: PIPPO_ID,
        receivers: [PAPERINO_ID,PLUTO_ID],
        message: message,
        subject: "Subject",
        data:    "timestamp"
    }
    const requestJSON = JSON.stringify(request)
    const requestHeaders =  await signRequest(requestJSON, PIPPO_EDDSA);
    const config = {
        headers: requestHeaders
    };

    const result = await axios.post(`${url}/send`, request, config);
    return result
}

const assertPostMany = async() => {
    for(let i=0; i<10; i++) {
        const res = await sendMessage(`Ciao a tutti ${i}`)

        console.assert(res.data.success)
        console.assert(res.data.count == 2)
    }
}

const readMessages = async(email, key) => {
    const request = {
        request_id: 42,
        receiver: email
    }
    const requestJSON = JSON.stringify(request)
    const requestHeaders =  await signRequest(requestJSON, key);
    const config = {
        headers: requestHeaders
    };

    const result = await axios.post(`${url}/read`, request, config);
    return result
}

const assertReadMany = async(email, key) => {
    const res = await readMessages(email, key)
    console.assert(res.data.success)
    res.data.messages.forEach((v, i) => {
        console.assert(v.message.startsWith("Ciao a tutti"))
        console.assert(v.subject == "Subject")
        console.assert(v.receivers.length == 2)
    })

}
await assertPostMany()
assertReadMany(PLUTO_ID, PLUTO_EDDSA)
assertReadMany(PAPERINO_ID, PAPERINO_EDDSA)
