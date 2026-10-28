import { useEffect, useState } from "react";
import { ActivityIndicator, KeyboardAvoidingView, Platform, SafeAreaView, ScrollView, StatusBar, StyleSheet, Text, TextInput, TouchableOpacity, View } from "react-native";
import * as SecureStore from "expo-secure-store";
import { AuthClient, AuthError } from "@arbeiter/auth-sdk";

const SESSION_KEY = "arbeiter.session.cookie";
const apiURL = process.env.EXPO_PUBLIC_AUTH_API_URL ?? "http://localhost:8080";
const auth = new AuthClient({
  baseURL: apiURL,
  fetchImpl: async (input, init) => {
    const headers = new Headers(init?.headers);
    const cookie = await SecureStore.getItemAsync(SESSION_KEY);
    if (cookie) headers.set("Cookie", cookie);
    const response = await fetch(input, { ...init, headers });
    const setCookie = response.headers.get("set-cookie");
    if (setCookie) {
      const pair = setCookie.split(";", 1)[0];
      if (pair === "session=" || /max-age=0/i.test(setCookie)) await SecureStore.deleteItemAsync(SESSION_KEY);
      else if (pair.startsWith("session=")) await SecureStore.setItemAsync(SESSION_KEY, pair);
    }
    return response;
  },
});

type User = { email?: string; name?: string; email_verified?: boolean };
export default function App() {
  const [user, setUser] = useState<User | null>(null);
  const [email, setEmail] = useState(""); const [password, setPassword] = useState(""); const [name, setName] = useState("");
  const [isSignUp, setIsSignUp] = useState(false); const [busy, setBusy] = useState(true); const [message, setMessage] = useState("");
  async function refresh() { try { setUser(await auth.me() as User); } catch { setUser(null); } }
  useEffect(() => { void refresh().finally(() => setBusy(false)); }, []);
  async function submit() {
    setBusy(true); setMessage("");
    try {
      if (isSignUp) await auth.signUp(email, password, name || undefined);
      else await auth.signIn(email, password);
      await refresh(); setMessage(isSignUp ? "Your account is ready." : "You are signed in.");
    } catch (error) { setMessage(error instanceof AuthError ? error.message : "Could not connect to the authentication service."); }
    finally { setBusy(false); }
  }
  async function signOut() { await auth.signOut(); await SecureStore.deleteItemAsync(SESSION_KEY); setUser(null); setMessage("You have signed out."); }
  return <SafeAreaView style={styles.safe}><StatusBar barStyle="dark-content"/><KeyboardAvoidingView style={styles.safe} behavior={Platform.OS === "ios" ? "padding" : undefined}><ScrollView contentContainerStyle={styles.page} keyboardShouldPersistTaps="handled">
    <View style={styles.top}><View style={styles.mark}><Text style={styles.markText}>a</Text></View><Text style={styles.wordmark}>arbeiter</Text><View style={styles.environment}><View style={styles.dot}/><Text style={styles.envText}>SECURE IDENTITY</Text></View></View>
    <View style={styles.hero}><Text style={styles.eyebrow}>YOUR ACCOUNT, IN YOUR HANDS</Text><Text style={styles.title}>{user ? "You’re all set." : isSignUp ? "Create your\naccount." : "Welcome\nback."}</Text><Text style={styles.subtitle}>{user ? "Your secure session is active on this device." : "Simple, secure access to your workspace."}</Text></View>
    <View style={styles.card}>{busy && !user ? <ActivityIndicator color="#5b49dc" style={{ marginVertical: 35 }}/> : user ? <View style={styles.account}><View style={styles.avatar}><Text style={styles.avatarText}>{user.name?.[0]?.toUpperCase() ?? "A"}</Text></View><Text style={styles.accountName}>{user.name || "Your account"}</Text><Text style={styles.accountEmail}>{user.email}</Text><View style={styles.verified}><View style={styles.dot}/><Text style={styles.verifiedText}>{user.email_verified ? "Email verified" : "Session authenticated"}</Text></View><TouchableOpacity style={styles.primary} onPress={() => void signOut()}><Text style={styles.primaryText}>Sign out</Text></TouchableOpacity></View> : <><Text style={styles.formTitle}>{isSignUp ? "Get started" : "Sign in"}</Text><Text style={styles.formHint}>{isSignUp ? "Set up your Arbeiter account." : "Enter your details to continue."}</Text>{isSignUp&&<><Text style={styles.label}>NAME <Text style={styles.optional}>OPTIONAL</Text></Text><TextInput style={styles.input} placeholder="Your name" value={name} onChangeText={setName} autoCapitalize="words"/></>}<Text style={styles.label}>EMAIL ADDRESS</Text><TextInput style={styles.input} placeholder="you@example.com" value={email} onChangeText={setEmail} keyboardType="email-address" autoCapitalize="none" autoComplete="email"/><Text style={styles.label}>PASSWORD</Text><TextInput style={styles.input} placeholder={isSignUp ? "At least 12 characters" : "Your password"} value={password} onChangeText={setPassword} secureTextEntry autoComplete={isSignUp ? "new-password" : "current-password"}/><TouchableOpacity style={styles.primary} onPress={() => void submit()} disabled={busy}><Text style={styles.primaryText}>{busy ? "Connecting…" : isSignUp ? "Create account" : "Continue securely"}</Text><Text style={styles.arrow}>→</Text></TouchableOpacity><View style={styles.switchRow}><Text style={styles.switchText}>{isSignUp ? "Already have an account?" : "New to Arbeiter?"}</Text><Text onPress={() => setIsSignUp(!isSignUp)} style={styles.switchAction}>{isSignUp ? "Sign in" : "Create account"}</Text></View></>}{message? <Text style={styles.message}>{message}</Text>:null}</View>
    <View style={styles.security}><Text style={styles.shield}>⌑</Text><View><Text style={styles.securityTitle}>Your session stays private</Text><Text style={styles.securityCopy}>Protected with encrypted device storage and secure cookies.</Text></View></View><Text style={styles.footer}>ARBEITER AUTH · IDENTITY INFRASTRUCTURE</Text>
  </ScrollView></KeyboardAvoidingView></SafeAreaView>;
}

const styles = StyleSheet.create({safe:{flex:1,backgroundColor:"#f7f7fc"},page:{flexGrow:1,paddingHorizontal:24,paddingTop:12,paddingBottom:24,maxWidth:520,width:"100%",alignSelf:"center"},top:{flexDirection:"row",alignItems:"center"},mark:{width:30,height:30,borderRadius:9,backgroundColor:"#5b49dc",alignItems:"center",justifyContent:"center"},markText:{color:"white",fontSize:19,fontWeight:"800"},wordmark:{fontSize:18,fontWeight:"800",color:"#172133",marginLeft:9},environment:{marginLeft:"auto",flexDirection:"row",alignItems:"center",gap:6},envText:{fontSize:8,color:"#8e98a8",letterSpacing:1,fontWeight:"700"},dot:{width:6,height:6,borderRadius:4,backgroundColor:"#25b982"},hero:{paddingTop:64,paddingBottom:25},eyebrow:{fontSize:9,letterSpacing:1.2,fontWeight:"700",color:"#6251d7"},title:{fontSize:43,lineHeight:48,fontWeight:"800",letterSpacing:-1.8,color:"#172133",marginTop:13},subtitle:{fontSize:13,lineHeight:20,color:"#7c8797",marginTop:12},card:{backgroundColor:"white",borderWidth:1,borderColor:"#eceef4",borderRadius:13,padding:20,shadowColor:"#252e4b",shadowOpacity:.05,shadowRadius:18,elevation:2},formTitle:{fontSize:17,fontWeight:"700",color:"#1a2434"},formHint:{fontSize:10,color:"#8b95a4",marginTop:4,marginBottom:20},label:{fontSize:8,letterSpacing:.8,color:"#647083",fontWeight:"700",marginBottom:6,marginTop:12},optional:{color:"#a1a9b5",fontWeight:"500"},input:{height:43,borderWidth:1,borderColor:"#e7e9f0",borderRadius:7,paddingHorizontal:12,fontSize:12,color:"#243044",backgroundColor:"#fff"},primary:{height:45,backgroundColor:"#5b49dc",borderRadius:7,marginTop:20,alignItems:"center",justifyContent:"center",flexDirection:"row"},primaryText:{color:"white",fontSize:11,fontWeight:"700"},arrow:{color:"white",position:"absolute",right:15},switchRow:{flexDirection:"row",justifyContent:"center",gap:5,marginTop:17},switchText:{fontSize:9,color:"#8d97a6"},switchAction:{fontSize:9,color:"#5b49dc",fontWeight:"700"},account:{alignItems:"center",paddingVertical:10},avatar:{width:55,height:55,borderRadius:28,backgroundColor:"#e9e6ff",alignItems:"center",justifyContent:"center"},avatarText:{fontSize:21,color:"#5946d6",fontWeight:"700"},accountName:{fontSize:15,fontWeight:"700",color:"#1a2434",marginTop:12},accountEmail:{fontSize:10,color:"#8490a0",marginTop:4},verified:{flexDirection:"row",alignItems:"center",gap:6,paddingVertical:9},verifiedText:{fontSize:9,color:"#568a73"},security:{flexDirection:"row",alignItems:"center",gap:12,marginTop:24,padding:14,backgroundColor:"#f0efff",borderRadius:9},shield:{width:27,height:27,textAlign:"center",textAlignVertical:"center",borderRadius:8,overflow:"hidden",backgroundColor:"white",color:"#6251d7",fontSize:17},securityTitle:{fontSize:10,color:"#384355",fontWeight:"700"},securityCopy:{fontSize:8,color:"#818b9a",marginTop:3},message:{fontSize:10,color:"#5b49dc",textAlign:"center",marginTop:13},footer:{fontSize:8,color:"#a1a9b5",letterSpacing:1,textAlign:"center",marginTop:"auto",paddingTop:35}});
