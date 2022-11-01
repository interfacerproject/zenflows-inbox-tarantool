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
//const url="https://gateway0.interfacer.dyne.org/inbox"

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
        content: {
            message: message,
            subject: "Subject",
            data:    "timestamp"
        }
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
        receiver: email,
        //only_unread: false,
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
        console.log(v)
        console.assert(v.content.message.startsWith("Ciao a tutti"))
        console.assert(v.content.subject == "Subject")
        console.assert(!v.read)
    })

}

const setMessage = async(message_id, receiver, read, key) => {
    const request = {
        message_id,
        receiver,
        read
    }
    const requestJSON = JSON.stringify(request)
    const requestHeaders =  await signRequest(requestJSON, key);
    const config = {
        headers: requestHeaders
    };

    const result = await axios.post(`${url}/set-read`, request, config);
    return result
}
//await assertPostMany()
assertReadMany(PLUTO_ID, PLUTO_EDDSA)
//assertReadMany(PAPERINO_ID, PAPERINO_EDDSA)
setMessage(230, PLUTO_ID, true, PLUTO_EDDSA)
