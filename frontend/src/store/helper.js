function getHeader(context, payload) {
    let method = payload.method || "GET"
    let req = {
        method: method,
        mode: "cors",
        credentials: "include",
        headers: {
            "Content-Type": "application/json",
            "Authorization": "Bearer " + context.state.accessToken
        },
    };
    if (method != 'GET') {
        req.body = JSON.stringify(payload.body)
    }
    return req
}

export { getHeader }