#!/usr/bin/env node
import sign from "./sign_graphql.mjs"
import { zencode_exec } from 'zenroom';
import axios from 'axios';

const PIPPO_EDDSA = "AEkxhh4aFV1eG88FY8LjZSMyJXgmynUdWzUPV6tCHwqn"
const PLUTO_EDDSA = "A7mSkKeAvAnDeeuWNW5TuBLnmKCLdyrJK652SZj2xmiP"
const PAPERINO_EDDSA = "88XLEXAkTdxdm4r8V5gYFPQxvqgMvWu4EHXKSMbXenzC"

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
        sender: "pippo@disney.com",
        receiver: ["paperino@dyne.org","pluto@dyne.org"],
        message: message,
        subject: "Subject",
        data:    "timestamp"
    }
    const requestJSON = JSON.stringify(request)
    const requestHeaders =  await signRequest(requestJSON, PIPPO_EDDSA);
    const config = {
        headers: requestHeaders
    };

    const result = await axios.post("http://localhost:8080/send", request, config);
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
        sender: email
    }
    const requestJSON = JSON.stringify(request)
    const requestHeaders =  await signRequest(requestJSON, key);
    const config = {
        headers: requestHeaders
    };

    const result = await axios.post("http://localhost:8080/read", request, config);
    return result
}

const assertReadMany = async(email, key) => {
    const res = await readMessages(email, key)
    console.assert(res.data.success)
    res.data.messages.forEach((v, i) => {
        console.assert(v.message.startsWith("Ciao a tutti"))
        console.assert(v.subject == "Subject")
    })

}
await assertPostMany()
assertReadMany("pluto@dyne.org", PLUTO_EDDSA)
assertReadMany("paperino@dyne.org", PAPERINO_EDDSA)
